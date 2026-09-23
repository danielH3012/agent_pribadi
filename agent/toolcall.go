package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	toolPkg "agentPribadi/tools"
)

type ToolCall struct {
	ID        string         `json:"id,omitempty"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
	RawArgs   string         `json:"raw_args,omitempty"`
}

// ExecuteTool mengeksekusi tool berdasarkan nama dan arguments berupa JSON string.
// Memetakan panggilan langsung ke function terkait di package tools.
func ExecuteTool(name string, argsJSON string) (any, error) {
	switch name {
	case "bash":
		var params struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
			return nil, fmt.Errorf("invalid arguments for bash: %w", err)
		}
		return toolPkg.Bash(params.Command)

	case "edit":
		var params struct {
			Path       string `json:"path"`
			FilePath   string `json:"file_path"`
			OldContent string `json:"old_content"`
			NewContent string `json:"new_content"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
			return nil, fmt.Errorf("invalid arguments for edit: %w", err)
		}
		targetPath := params.Path
		if targetPath == "" {
			targetPath = params.FilePath
		}
		return toolPkg.EditFile(targetPath, params.NewContent, params.OldContent)

	case "glob":
		var params struct {
			Pattern string `json:"pattern"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
			return nil, fmt.Errorf("invalid arguments for glob: %w", err)
		}
		return toolPkg.Glob(params.Pattern)

	case "grep":
		var params struct {
			Pattern      string `json:"pattern"`
			Path         string `json:"path"`
			SearchPath   string `json:"search_path"`
			ContextLines int    `json:"context_lines"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
			return nil, fmt.Errorf("invalid arguments for grep: %w", err)
		}
		targetPath := params.Path
		if targetPath == "" {
			targetPath = params.SearchPath
		}
		if params.ContextLines > 0 {
			return toolPkg.GrepWithContext(params.Pattern, targetPath, params.ContextLines)
		}
		return toolPkg.Grep(params.Pattern, targetPath)

	case "multi_edit", "multiEdit":
		var params struct {
			Operations []toolPkg.EditResult `json:"operations"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
			return nil, fmt.Errorf("invalid arguments for multi_edit: %w", err)
		}
		return toolPkg.MultiEdit(params.Operations)

	case "read":
		var params struct {
			Path     string `json:"path"`
			FilePath string `json:"file_path"`
			Limit    int    `json:"limit"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
			return nil, fmt.Errorf("invalid arguments for read: %w", err)
		}
		targetPath := params.Path
		if targetPath == "" {
			targetPath = params.FilePath
		}
		if params.Limit > 0 {
			return toolPkg.ReadFile(targetPath, params.Limit)
		}
		return toolPkg.ReadFile(targetPath)

	case "write":
		var params struct {
			Path     string `json:"path"`
			FilePath string `json:"file_path"`
			Content  string `json:"content"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &params); err != nil {
			return nil, fmt.Errorf("invalid arguments for write: %w", err)
		}
		targetPath := params.Path
		if targetPath == "" {
			targetPath = params.FilePath
		}
		return toolPkg.WriteFile(targetPath, params.Content)

	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// ProcessToolCalls mengeksekusi daftar ToolCall dan mengembalikan string hasil untuk dikirim ke LLM.
func ProcessToolCalls(calls []ToolCall) []string {
	var outputs []string
	for _, tc := range calls {

		argsJSON := tc.RawArgs
		if argsJSON == "" && tc.Arguments != nil {
			b, _ := json.Marshal(tc.Arguments)
			argsJSON = string(b)
		}

		result, err := ExecuteTool(tc.Name, argsJSON)
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
