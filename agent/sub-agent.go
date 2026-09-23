package agent

import (
	"context"
	"fmt"
	"strings"
)

func FileSummerizer(ctx context.Context, query string, tools []ToolDefinition) (string, error) {
	if len(tools) == 0 {
		tools = []ToolDefinition{GlobTool, GrepTool, ReadTool}
	}

	var toolGuide strings.Builder
	for _, t := range tools {
		toolGuide.WriteString(fmt.Sprintf("- %s: %s\n", t.Function.Name, t.Function.Description))
	}

	systemprompt := fmt.Sprintf(`
		You are a File Summarizer Agent.
		Your goal is to analyze the content of files based on the user's request and provide a concise summary of the relevant information.
		Analyze the user request and conversation history to select and execute the necessary tools accurately.

		**Guidelines:**
			- If you encounter identifiers, preserve their exact format.
			- Provide direct and concise tool calls without conversational pleasantries.
			- Populate all required tool arguments based on the user request and context.

		Available Tools Guide:
		%s

		Execution Rules:
			1. **Thinking (<thought>...</thought>)**: Analyze the user's intent and determine what data or tools are required.
			2. **Tool Execution**: Call only authorized tools available in the Available Tools Guide.
			3. **No Tools Needed**: If no tools are required (e.g. conversational greetings or questions that do not need external actions), call 'no_tools'.

		Output format:
			If tools are needed:
				<thought>Brief reasoning on why the tool is selected and what arguments are used</thought>
				<tool_call>
					{"name": "tool_name", "arguments": {...}}
				</tool_call>

			If NO tools are needed:
				<thought>Brief reasoning</thought>
				<tool_call>
					{"name": "no_tools", "arguments": {"reason": "greeting or no tool required"}}
				</tool_call>
	`, toolGuide.String())
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

		calls := ParseToolCalls(res.RawOutput)

		var validCalls []ToolCall
		for _, tc := range calls {
			if tc.Name != "no_tools" && tc.Name != "no_tool" {
				validCalls = append(validCalls, tc)
			}
		}

		if len(validCalls) == 0 {
			cleanText := strings.TrimSpace(res.RawOutput)
			if cleanText != "" {
				return cleanText, nil
			}
		}

		messages = append(messages, Message{Role: "assistant", Content: res.RawOutput})

		toolOutputs := ProcessToolCalls(validCalls)

		messages = append(messages, Message{
			Role:    "user",
			Content: fmt.Sprintf("Tool Results:\n%s\n\nSilakan lanjutkan analisis atau berikan rangkuman akhir Anda.", strings.Join(toolOutputs, "\n\n")),
		})
	}

	messages = append(messages, Message{Role: "user", Content: "Silakan berikan rangkuman akhir Anda sekarang berdasarkan informasi yang telah didapatkan."})
	finalRes, err := ChatGenerate(ctx, messages, nil, 262144, GeneratorModel, 2)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(finalRes.RawOutput), nil
}

func CodeFixer(ctx context.Context, query string, tools []ToolDefinition) (string, error) {
	if len(tools) == 0 {
		tools = []ToolDefinition{GlobTool, GrepTool, ReadTool, EditTool, BashTool, MultiEditTool}
	}

	var toolGuide strings.Builder
	for _, t := range tools {
		toolGuide.WriteString(fmt.Sprintf("- %s: %s\n", t.Function.Name, t.Function.Description))
	}

	systemprompt := fmt.Sprintf(`
		You are a Code Fixer Agent.
		Your goal is to fix and update code based on the user's request.
		Guidelines:
			- If you encounter identifiers, preserve their exact format.
			- Provide direct and concise tool calls without conversational pleasantries.
			- Populate all required tool arguments based on the user request and context.

		Available Tools Guide:
		%s

		Execution Rules:
			1. **Thinking (<thought>...</thought>)**: Analyze the user's intent and determine what data or tools are required.
			2. **Tool Execution**: Call only authorized tools available in the Available Tools Guide.
			3. **No Tools Needed**: If no tools are required (e.g. conversational greetings or questions that do not need external actions), call 'no_tools'.

		Output format:
			If tools are needed:
				<thought>Brief reasoning on why the tool is selected and what arguments are used</thought>
				<tool_call>
					{"name": "tool_name", "arguments": {...}}
				</tool_call>

			If NO tools are needed:
				<thought>Brief reasoning</thought>
				<tool_call>
					{"name": "no_tools", "arguments": {"reason": "greeting or no tool required"}}
				</tool_call>
	`, toolGuide.String())
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

		calls := ParseToolCalls(res.RawOutput)

		var validCalls []ToolCall
		for _, tc := range calls {
			if tc.Name != "no_tools" && tc.Name != "no_tool" {
				validCalls = append(validCalls, tc)
			}
		}

		if len(validCalls) == 0 {
			cleanText := strings.TrimSpace(res.RawOutput)
			if cleanText != "" {
				return cleanText, nil
			}
		}

		messages = append(messages, Message{Role: "assistant", Content: res.RawOutput})

		toolOutputs := ProcessToolCalls(validCalls)

		messages = append(messages, Message{
			Role:    "user",
			Content: fmt.Sprintf("Tool Results:\n%s\n\nSilakan lanjutkan analisis atau berikan rangkuman akhir Anda.", strings.Join(toolOutputs, "\n\n")),
		})
	}

	messages = append(messages, Message{Role: "user", Content: "Silakan berikan rangkuman akhir Anda sekarang berdasarkan informasi yang telah didapatkan."})
	finalRes, err := ChatGenerate(ctx, messages, nil, 262144, GeneratorModel, 2)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(finalRes.RawOutput), nil
}

func SoftwareAgent(ctx context.Context, query string, tools []ToolDefinition) (string, error) {
	if len(tools) == 0 {
		tools = []ToolDefinition{GlobTool, GrepTool, ReadTool, EditTool, BashTool, MultiEditTool, WriteTool}
	}

	var toolGuide strings.Builder
	for _, t := range tools {
		toolGuide.WriteString(fmt.Sprintf("- %s: %s\n", t.Function.Name, t.Function.Description))
	}

	systemprompt := fmt.Sprintf(`
		You are a program maker.
		Your goal is to make and update software/program/script based on the user's request.
		Guidelines:
			- If you encounter identifiers, preserve their exact format.
			- Provide direct and concise tool calls without conversational pleasantries.
			- Populate all required tool arguments based on the user request and context.

		Available Tools Guide:
		%s

		Execution Rules:
			1. **Thinking (<thought>...</thought>)**: Analyze the user's intent and determine what data or tools are required.
			2. **Tool Execution**: Call only authorized tools available in the Available Tools Guide.
			3. **No Tools Needed**: If no tools are required (e.g. conversational greetings or questions that do not need external actions), call 'no_tools'.

		Output format:
			If tools are needed:
				<thought>Brief reasoning on why the tool is selected and what arguments are used</thought>
				<tool_call>
					{"name": "tool_name", "arguments": {...}}
				</tool_call>

			If NO tools are needed:
				<thought>Brief reasoning</thought>
				<tool_call>
					{"name": "no_tools", "arguments": {"reason": "greeting or no tool required"}}
				</tool_call>
	`, toolGuide.String())
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

		calls := ParseToolCalls(res.RawOutput)

		var validCalls []ToolCall
		for _, tc := range calls {
			if tc.Name != "no_tools" && tc.Name != "no_tool" {
				validCalls = append(validCalls, tc)
			}
		}

		if len(validCalls) == 0 {
			cleanText := strings.TrimSpace(res.RawOutput)
			if cleanText != "" {
				return cleanText, nil
			}
		}

		messages = append(messages, Message{Role: "assistant", Content: res.RawOutput})

		toolOutputs := ProcessToolCalls(validCalls)

		messages = append(messages, Message{
			Role:    "user",
			Content: fmt.Sprintf("Tool Results:\n%s\n\nSilakan lanjutkan analisis atau berikan rangkuman akhir Anda.", strings.Join(toolOutputs, "\n\n")),
		})
	}

	messages = append(messages, Message{Role: "user", Content: "Silakan berikan rangkuman akhir Anda sekarang berdasarkan informasi yang telah didapatkan."})
	finalRes, err := ChatGenerate(ctx, messages, nil, 262144, GeneratorModel, 2)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(finalRes.RawOutput), nil
}

func TestingAgent(ctx context.Context, query string, tools []ToolDefinition) (string, error) {
	if len(tools) == 0 {
		tools = []ToolDefinition{GlobTool, GrepTool, ReadTool, BashTool, WriteTool, EditTool}
	}

	var toolGuide strings.Builder
	for _, t := range tools {
		toolGuide.WriteString(fmt.Sprintf("- %s: %s\n", t.Function.Name, t.Function.Description))
	}

	systemprompt := fmt.Sprintf(`
		You are a testing agent.
		Your goal is to test program/script based on the user's request.
		You need to make a test file and run that program with bash tool.
		Guidelines:
			- If you encounter identifiers, preserve their exact format.
			- Provide direct and concise tool calls without conversational pleasantries.
			- Populate all required tool arguments based on the user request and context.

		Available Tools Guide:
		%s

		Execution Rules:
			1. **Thinking (<thought>...</thought>)**: Analyze the user's intent and determine what data or tools are required.
			2. **Tool Execution**: Call only authorized tools available in the Available Tools Guide.
			3. **No Tools Needed**: If no tools are required (e.g. conversational greetings or questions that do not need external actions), call 'no_tools'.

		Output format:
			If tools are needed:
				<thought>Brief reasoning on why the tool is selected and what arguments are used</thought>
				<tool_call>
					{"name": "tool_name", "arguments": {...}}
				</tool_call>

			If NO tools are needed:
				<thought>Brief reasoning</thought>
				<tool_call>
					{"name": "no_tools", "arguments": {"reason": "greeting or no tool required"}}
				</tool_call>
	`, toolGuide.String())
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

		calls := ParseToolCalls(res.RawOutput)

		var validCalls []ToolCall
		for _, tc := range calls {
			if tc.Name != "no_tools" && tc.Name != "no_tool" {
				validCalls = append(validCalls, tc)
			}
		}

		if len(validCalls) == 0 {
			cleanText := strings.TrimSpace(res.RawOutput)
			if cleanText != "" {
				return cleanText, nil
			}
		}

		messages = append(messages, Message{Role: "assistant", Content: res.RawOutput})

		toolOutputs := ProcessToolCalls(validCalls)

		messages = append(messages, Message{
			Role:    "user",
			Content: fmt.Sprintf("Tool Results:\n%s\n\nSilakan lanjutkan analisis atau berikan rangkuman akhir Anda.", strings.Join(toolOutputs, "\n\n")),
		})
	}

	messages = append(messages, Message{Role: "user", Content: "Silakan berikan rangkuman akhir Anda sekarang berdasarkan informasi yang telah didapatkan."})
	finalRes, err := ChatGenerate(ctx, messages, nil, 262144, GeneratorModel, 2)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(finalRes.RawOutput), nil
}

func ReviewerAgent(ctx context.Context, query string, tools []ToolDefinition) (string, error) {
	if len(tools) == 0 {
		tools = []ToolDefinition{GlobTool, GrepTool, ReadTool, EditTool, BashTool, MultiEditTool, WriteTool}
	}

	var toolGuide strings.Builder
	for _, t := range tools {
		toolGuide.WriteString(fmt.Sprintf("- %s: %s\n", t.Function.Name, t.Function.Description))
	}

	systemprompt := fmt.Sprintf(`
		You are a reviewer agent.
		Your goal is to review code and provide feedback without making changes.
		Guidelines:
			- If you encounter identifiers, preserve their exact format.
			- Provide direct and concise tool calls without conversational pleasantries.
			- Populate all required tool arguments based on the user request and context.

		Available Tools Guide:
		%s

		Execution Rules:
			1. **Thinking (<thought>...</thought>)**: Analyze the user's intent and determine what data or tools are required.
			2. **Tool Execution**: Call only authorized tools available in the Available Tools Guide.
			3. **No Tools Needed**: If no tools are required (e.g. conversational greetings or questions that do not need external actions), call 'no_tools'.

		Output format:
			If tools are needed:
				<thought>Brief reasoning on why the tool is selected and what arguments are used</thought>
				<tool_call>
					{"name": "tool_name", "arguments": {...}}
				</tool_call>

			If NO tools are needed:
				<thought>Brief reasoning</thought>
				<tool_call>
					{"name": "no_tools", "arguments": {"reason": "greeting or no tool required"}}
				</tool_call>
	`, toolGuide.String())
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

		calls := ParseToolCalls(res.RawOutput)

		var validCalls []ToolCall
		for _, tc := range calls {
			if tc.Name != "no_tools" && tc.Name != "no_tool" {
				validCalls = append(validCalls, tc)
			}
		}

		if len(validCalls) == 0 {
			cleanText := strings.TrimSpace(res.RawOutput)
			if cleanText != "" {
				return cleanText, nil
			}
		}

		messages = append(messages, Message{Role: "assistant", Content: res.RawOutput})

		toolOutputs := ProcessToolCalls(validCalls)

		messages = append(messages, Message{
			Role:    "user",
			Content: fmt.Sprintf("Tool Results:\n%s\n\nSilakan lanjutkan analisis atau berikan rangkuman akhir Anda.", strings.Join(toolOutputs, "\n\n")),
		})
	}

	messages = append(messages, Message{Role: "user", Content: "Silakan berikan rangkuman akhir Anda sekarang berdasarkan informasi yang telah didapatkan."})
	finalRes, err := ChatGenerate(ctx, messages, nil, 262144, GeneratorModel, 2)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(finalRes.RawOutput), nil
}

func DocumenterAgent(ctx context.Context, query string, tools []ToolDefinition) (string, error) {
	if len(tools) == 0 {
		tools = []ToolDefinition{GlobTool, GrepTool, ReadTool, WriteTool}
	}

	var toolGuide strings.Builder
	for _, t := range tools {
		toolGuide.WriteString(fmt.Sprintf("- %s: %s\n", t.Function.Name, t.Function.Description))
	}

	systemprompt := fmt.Sprintf(`
		You are a documenter agent.
		Your goal is to make a documentation based on the user's project and request.
		Guidelines:
			- If you encounter identifiers, preserve their exact format.
			- Provide direct and concise tool calls without conversational pleasantries.
			- Populate all required tool arguments based on the user request and context.

		Available Tools Guide:
		%s

		Execution Rules:
			1. **Thinking (<thought>...</thought>)**: Analyze the user's intent and determine what data or tools are required.
			2. **Tool Execution**: Call only authorized tools available in the Available Tools Guide.
			3. **No Tools Needed**: If no tools are required (e.g. conversational greetings or questions that do not need external actions), call 'no_tools'.

		Output format:
			If tools are needed:
				<thought>Brief reasoning on why the tool is selected and what arguments are used</thought>
				<tool_call>
					{"name": "tool_name", "arguments": {...}}
				</tool_call>

			If NO tools are needed:
				<thought>Brief reasoning</thought>
				<tool_call>
					{"name": "no_tools", "arguments": {"reason": "greeting or no tool required"}}
				</tool_call>
	`, toolGuide.String())
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

		calls := ParseToolCalls(res.RawOutput)

		var validCalls []ToolCall
		for _, tc := range calls {
			if tc.Name != "no_tools" && tc.Name != "no_tool" {
				validCalls = append(validCalls, tc)
			}
		}

		if len(validCalls) == 0 {
			cleanText := strings.TrimSpace(res.RawOutput)
			if cleanText != "" {
				return cleanText, nil
			}
		}

		messages = append(messages, Message{Role: "assistant", Content: res.RawOutput})

		toolOutputs := ProcessToolCalls(validCalls)

		messages = append(messages, Message{
			Role:    "user",
			Content: fmt.Sprintf("Tool Results:\n%s\n\nSilakan lanjutkan analisis atau berikan rangkuman akhir Anda.", strings.Join(toolOutputs, "\n\n")),
		})
	}

	messages = append(messages, Message{Role: "user", Content: "Silakan berikan rangkuman akhir Anda sekarang berdasarkan informasi yang telah didapatkan."})
	finalRes, err := ChatGenerate(ctx, messages, nil, 262144, GeneratorModel, 2)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(finalRes.RawOutput), nil
}
