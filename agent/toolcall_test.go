package agent

import (
	"context"
	"testing"

	"agentPribadi/mcp"
	"agentPribadi/mcp/tools"
)

func TestUnifiedMCPTools(t *testing.T) {
	ctx := context.Background()
	tools.AutoApprove = true

	// 1. Verifikasi Default Server memuat 7 built-in tools
	server := mcp.DefaultServer()
	tools := GetAgentTools(server)

	if len(tools) != 7 {
		t.Fatalf("expected 7 built-in tools, got %d", len(tools))
	}

	// Pastikan tool bash, edit, glob, grep, read, write, multi_edit ada
	toolMap := make(map[string]bool)
	for _, tool := range tools {
		toolMap[tool.Function.Name] = true
	}

	expected := []string{"bash", "edit", "glob", "grep", "multi_edit", "read", "write"}
	for _, name := range expected {
		if !toolMap[name] {
			t.Errorf("missing expected tool: %s", name)
		}
	}

	// 2. Verifikasi user bisa menambahkan custom tool mereka sendiri
	type PingInput struct {
		Message string `json:"message" desc:"Ping message"`
	}
	customPingTool := mcp.Func("ping", "A simple ping test tool", func(ctx context.Context, in PingInput) (any, error) {
		return "pong: " + in.Message, nil
	})

	if err := server.Add(customPingTool); err != nil {
		t.Fatalf("failed to add custom tool: %v", err)
	}

	toolsWithCustom := GetAgentTools(server)
	if len(toolsWithCustom) != 8 {
		t.Fatalf("expected 8 tools after adding custom tool, got %d", len(toolsWithCustom))
	}

	// 3. Verifikasi ExecuteTool mengeksekusi custom tool via server.Call
	res, err := ExecuteTool(ctx, "ping", `{"message": "hello world"}`, server)
	if err != nil {
		t.Fatalf("ExecuteTool error on custom tool: %v", err)
	}
	if res != "pong: hello world" {
		t.Fatalf("unexpected custom tool result: %v", res)
	}

	// 4. Verifikasi ExecuteTool mengeksekusi built-in tool (misal: glob)
	globRes, err := ExecuteTool(ctx, "glob", `{"pattern": "*.go", "path": "."}`, server)
	if err != nil {
		t.Fatalf("ExecuteTool error on built-in tool: %v", err)
	}
	if globRes == nil {
		t.Fatalf("expected non-nil glob result")
	}
}
