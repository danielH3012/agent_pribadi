package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func TaskMaker(ctx context.Context, query string, tools []ToolDefinition) (string, error) {
	systemprompt := fmt.Sprintf(`
		You are a taskmaker agent.
		Your goal is to change user prompt into a good structured instruction stages that are detail and easily executed by the sub-agents.

		Output format:
			1. stages one \n
			2. stages two \n
			3. stages three \n
			4. .......
	`)
	messages := []Message{
		{
			Role:    "system",
			Content: systemprompt,
		},
		{
			Role:    "user",
			Content: query,
		},
	}

	maxIterations := 5
	for i := 0; i < maxIterations; i++ {
		res, err := ChatGenerate(ctx, messages, tools, 262144, GeneratorModel, 2)
		if err != nil {
			return "", fmt.Errorf("ChatGenerate error: %w", err)
		}

		messages = append(messages, Message{Role: "assistant", Content: res.RawOutput})
	}

	messages = append(messages, Message{Role: "user", Content: "Silakan berikan rangkuman akhir Anda sekarang berdasarkan informasi yang telah didapatkan."})
	finalRes, err := ChatGenerate(ctx, messages, nil, 262144, GeneratorModel, 2)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(finalRes.RawOutput), nil
}

func Orchestrator(ctx context.Context, query string, tools []ToolDefinition, prompt string) (string, error) {
	// 1. Panggil TaskMaker untuk membuat tahapan kerja (stages)
	stages, err := TaskMaker(ctx, query, tools)
	if err != nil {
		return "", fmt.Errorf("TaskMaker error: %w", err)
	}
	fmt.Printf("\n=== [STAGES DARI TASKMAKER] ===\n%s\n===============================\n", stages)
	// 2. Orchestrator membaca stages, menentukan sub-agent yang tepat, dan menyusun query khusus
	systemPrompt := prompt
	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{
			Role: "user",
			Content: fmt.Sprintf("User Request:\n%s\n\nStages from TaskMaker:\n%s\n\nSilakan tentukan sub-agent yang sesuai untuk tiap stage dan rumuskan query-nya.",
				query, stages),
		},
	}
	maxIterations := 100
	for i := 0; i < maxIterations; i++ {
		res, err := ChatGenerate(ctx, messages, SubAgentTools, 262144, GeneratorModel, 2)
		if err != nil {
			return "", err
		}
		calls := ParseToolCalls(res.RawOutput)
		if len(calls) == 0 {
			// Selesai jika Orchestrator tidak memanggil tool lagi
			return strings.TrimSpace(res.RawOutput), nil
		}
		messages = append(messages, Message{Role: "assistant", Content: res.RawOutput})
		// Eksekusi sub-agent yang dipilih oleh Orchestrator
		var subReports []string
		for _, tc := range calls {
			fmt.Printf("\n[Orchestrator] Memilih: %s\n", tc.Name)
			report, err := dispatchSubAgent(ctx, tc.Name, tc.RawArgs)
			if err != nil {
				subReports = append(subReports, fmt.Sprintf("Error from %s: %v", tc.Name, err))
			} else {
				subReports = append(subReports, fmt.Sprintf("Report from %s:\n%s", tc.Name, report))
			}
		}
		messages = append(messages, Message{
			Role: "user",
			Content: fmt.Sprintf("Hasil Sub-Agent:\n%s\n\nSilakan lanjutkan ke stage berikutnya atau berikan kesimpulan akhir jika semua stage selesai.",
				strings.Join(subReports, "\n\n---\n\n")),
		})
	}
	return "Batas iterasi orchestrator tercapai.", nil
}

// dispatchSubAgent menjalankan sub-agent sesuai pilihan Orchestrator
func dispatchSubAgent(ctx context.Context, toolName string, argsJSON string) (string, error) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	switch toolName {
	case "subAgent":
		return subAgent(ctx, params.Query, nil)
	default:
		return "", fmt.Errorf("sub-agent tidak dikenal: %s", toolName)
	}
}
