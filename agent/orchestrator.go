package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"agentPribadi/controller"
	"agentPribadi/mcp"
)

// OrchestratorTools defines the tools available to Orchestrator (namely, subAgent).
var OrchestratorTools = []ToolDefinition{
	{
		Type: "function",
		Function: ToolFunction{
			Name:        "subAgent",
			Description: "Delegate a specific task or stage to a sub-agent equipped with MCP tools (bash, file edit/read/write, grep, custom tools) to execute and return findings.",
			Parameters: &FunctionParams{
				Type: "object",
				Properties: map[string]PropertySchema{
					"query": {
						Type:        "string",
						Description: "The detailed instruction and objective for the sub-agent to execute.",
					},
				},
				Required:             []string{"query"},
				AdditionalProperties: false,
			},
		},
	},
}

// TaskMaker breaks down user query into sequential structured stages.
func TaskMaker(ctx context.Context, query string, tools []ToolDefinition) (string, error) {
	var toolInfo strings.Builder
	if len(tools) > 0 {
		toolInfo.WriteString("\nAvailable sub-agent tools that can be utilized:\n")
		for _, t := range tools {
			toolInfo.WriteString(fmt.Sprintf("- %s: %s\n", t.Function.Name, t.Function.Description))
		}
	}

	systemprompt := fmt.Sprintf(`You are a taskmaker agent.
Your goal is to change the user prompt into structured sequential instruction stages that are detailed and easily executed by sub-agents.
%s
Output format:
1. Stage one: [Brief title and clear instruction]
2. Stage two: [Brief title and clear instruction]
3. Stage three: [Brief title and clear instruction]
...`, toolInfo.String())

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

	// TaskMaker only needs a single prompt-generation pass
	res, err := ChatGenerate(ctx, messages, nil, 4096, GeneratorModel, 2)
	if err != nil {
		return "", fmt.Errorf("TaskMaker error: %w", err)
	}

	return strings.TrimSpace(res.RawOutput), nil
}

// Orchestrator coordinates the execution of stages by delegating tasks to sub-agents.
func Orchestrator(ctx context.Context, query string, tools []ToolDefinition, prompt string, mcpServer *mcp.Server) (string, error) {
	if mcpServer == nil {
		mcpServer = mcp.DefaultServer()
	}
	if len(tools) == 0 {
		tools = GetAgentTools(mcpServer)
	}

	subAgentCfg := controller.NewSubAgent()
	agent_prompt := subAgentCfg.Prompt

	// 1. Panggil TaskMaker untuk membuat tahapan kerja (stages)
	stages, err := TaskMaker(ctx, query, tools)
	if err != nil {
		return "", fmt.Errorf("TaskMaker error: %w", err)
	}
	fmt.Printf("\n=== [STAGES DARI TASKMAKER] ===\n%s\n===============================\n", stages)

	// 2. Susun system prompt Orchestrator dengan panduan jelas memanggil subAgent
	var sb strings.Builder
	if strings.TrimSpace(prompt) != "" {
		sb.WriteString(strings.TrimSpace(prompt) + "\n\n")
	} else {
		sb.WriteString("You are an Orchestrator agent.\nYour goal is to coordinate and solve the user's request step-by-step according to the stages planned by TaskMaker.\n\n")
	}

	sb.WriteString(`Execution Rules:
1. You MUST delegate each stage requiring work, searches, commands, code analysis, or file edits to the 'subAgent' tool.
2. Formulate clear, self-contained queries for the sub-agent for each stage.
3. Review the report returned by the sub-agent before moving to the next stage.
4. Output format to call sub-agent:
   <thought>Reasoning about which stage to execute now and why</thought>
   <tool_call>
   {"name": "subAgent", "arguments": {"query": "detailed instructions for the sub-agent"}}
   </tool_call>
5. Only when ALL stages are completed and you have all needed information, output your final comprehensive answer directly to the user WITHOUT any <tool_call> tag.`)

	systemPrompt := sb.String()

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{
			Role: "user",
			Content: fmt.Sprintf("User Request:\n%s\n\nStages from TaskMaker:\n%s\n\nSilakan mulai eksekusi stage pertama dengan memanggil subAgent.",
				query, stages),
		},
	}

	maxIterations := 25
	for i := 0; i < maxIterations; i++ {
		res, err := ChatGenerate(ctx, messages, OrchestratorTools, 262144, GeneratorModel, 2)
		if err != nil {
			return "", err
		}

		calls := ParseToolCalls(res.RawOutput)
		if len(calls) == 0 {
			// Selesai jika Orchestrator tidak memanggil tool lagi (sudah memberikan rangkuman/jawaban akhir)
			return strings.TrimSpace(res.RawOutput), nil
		}

		messages = append(messages, Message{Role: "assistant", Content: res.RawOutput})

		// Eksekusi sub-agent yang dipilih oleh Orchestrator
		var subReports []string
		for _, tc := range calls {
			fmt.Printf("\n[Orchestrator] Memilih: %s\n", tc.Name)
			report, err := dispatchSubAgent(ctx, tc.Name, tc.RawArgs, tools, mcpServer, agent_prompt)
			if err != nil {
				subReports = append(subReports, fmt.Sprintf("Error from %s: %v", tc.Name, err))
			} else {
				subReports = append(subReports, fmt.Sprintf("Report from %s:\n%s", tc.Name, report))
			}
		}

		messages = append(messages, Message{
			Role: "user",
			Content: fmt.Sprintf("Hasil Sub-Agent:\n%s\n\nSilakan lanjutkan ke stage berikutnya dengan memanggil subAgent, atau berikan kesimpulan akhir jika semua stage telah selesai.",
				strings.Join(subReports, "\n\n---\n\n")),
		})
	}
	return "Batas iterasi orchestrator tercapai.", nil
}

// dispatchSubAgent menjalankan sub-agent sesuai pilihan Orchestrator
func dispatchSubAgent(ctx context.Context, toolName string, argsJSON string, tools []ToolDefinition, mcpServer *mcp.Server, agent_prompt string) (string, error) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	switch toolName {
	case "subAgent":
		return subAgent(ctx, params.Query, tools, mcpServer, agent_prompt)
	default:
		return "", fmt.Errorf("sub-agent tidak dikenal: %s", toolName)
	}
}
