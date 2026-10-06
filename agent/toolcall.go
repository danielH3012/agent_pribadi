package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"agentPribadi/mcp"
)

// ToolDefinition represents a tool specification compatible with OpenAI/LLM function calling.
type ToolDefinition struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction describes a function and its expected JSON parameters.
type ToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

// FunctionParams is the JSON Schema representation for function arguments.
type FunctionParams struct {
	Type                 string                    `json:"type"`
	Properties           map[string]PropertySchema `json:"properties,omitempty"`
	Required             []string                  `json:"required,omitempty"`
	AdditionalProperties bool                      `json:"additionalProperties"`
}

// PropertySchema describes a single property schema inside parameters.
type PropertySchema struct {
	Type        string                    `json:"type"`
	Description string                    `json:"description,omitempty"`
	Items       *PropertySchema           `json:"items,omitempty"`
	Properties  map[string]PropertySchema `json:"properties,omitempty"`
	Required    []string                  `json:"required,omitempty"`
}

// GetAgentTools extracts and converts all tools registered in the unified MCP server
// into OpenAI-compatible ToolDefinitions for LLM function calling.
func GetAgentTools(mcpServer *mcp.Server) []ToolDefinition {
	if mcpServer == nil {
		mcpServer = mcp.DefaultServer()
	}

	var definitions []ToolDefinition
	for _, st := range mcpServer.Tools() {
		var params any
		if len(st.Tool.RawInputSchema) > 0 {
			var rawMap any
			if err := json.Unmarshal(st.Tool.RawInputSchema, &rawMap); err == nil {
				params = rawMap
			} else {
				params = st.Tool.RawInputSchema
			}
		} else {
			params = st.Tool.InputSchema
		}

		definitions = append(definitions, ToolDefinition{
			Type: "function",
			Function: ToolFunction{
				Name:        st.Tool.Name,
				Description: st.Tool.Description,
				Parameters:  params,
			},
		})
	}
	return definitions
}

// GetTools returns the default tool definitions from DefaultServer.
func GetTools() []ToolDefinition {
	return GetAgentTools(nil)
}

// Tools contains built-in tool definitions for backwards compatibility.
var Tools = GetTools()

type ToolCall struct {
	ID        string         `json:"id,omitempty"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
	RawArgs   string         `json:"raw_args,omitempty"`
}

// ExecuteTool mengeksekusi tool langsung melalui Unified MCP Server.
func ExecuteTool(ctx context.Context, name string, argsJSON string, mcpServer *mcp.Server) (any, error) {
	if mcpServer == nil {
		mcpServer = mcp.DefaultServer()
	}
	return mcpServer.Call(ctx, name, json.RawMessage(argsJSON))
}

// ProcessToolCalls mengeksekusi daftar ToolCall melalui Unified MCP Server dan mengembalikan output untuk LLM.
func ProcessToolCalls(ctx context.Context, calls []ToolCall, mcpServer *mcp.Server) []string {
	var outputs []string
	for _, tc := range calls {
		argsJSON := tc.RawArgs
		if argsJSON == "" && tc.Arguments != nil {
			b, _ := json.Marshal(tc.Arguments)
			argsJSON = string(b)
		}

		result, err := ExecuteTool(ctx, tc.Name, argsJSON, mcpServer)
		if err != nil {
			outputs = append(outputs, fmt.Sprintf("Tool %s error: %v", tc.Name, err))
		} else {
			resBytes, _ := json.Marshal(result)
			outputs = append(outputs, fmt.Sprintf("Tool %s result:\n%s", tc.Name, string(resBytes)))
		}
	}
	return outputs
}

// ParseToolCalls mengekstrak tool calls dari output teks jika model menghasilkan format <tool_call>...</tool_call>.
func ParseToolCalls(text string) []ToolCall {
	var calls []ToolCall
	startTag := "<tool_call>"
	endTag := "</tool_call>"

	curr := text
	for {
		start := strings.Index(curr, startTag)
		if start == -1 {
			break
		}
		end := strings.Index(curr[start:], endTag)
		if end == -1 {
			break
		}
		raw := strings.TrimSpace(curr[start+len(startTag) : start+end])
		var tc ToolCall
		if err := json.Unmarshal([]byte(raw), &tc); err == nil && tc.Name != "" {
			tc.RawArgs = raw
			calls = append(calls, tc)
		}
		curr = curr[start+end+len(endTag):]
	}
	return calls
}
