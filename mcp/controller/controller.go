package controller

import (
	"agentPribadi/mcp/tools"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func Run() error {
	s := server.NewMCPServer("qtera-inventory-mcp", "2.0.0", server.WithToolCapabilities(true))

	s.AddTools(
		server.ServerTool{
			Tool: mcp.NewTool("bash",
				mcp.WithDescription("Execute bash command"),
				mcp.WithString("command", mcp.Required(), mcp.Description("Command to run")),
			),
			Handler: tools.BashHandler, // no parentheses
		},

		server.ServerTool{
			Tool: mcp.NewTool("edit",
				mcp.WithDescription("Replace exact text in a file"),
				mcp.WithString("file_path", mcp.Required(), mcp.Description("Path to the file")),
				mcp.WithString("old_content", mcp.Required(), mcp.Description("Exact text to replace")),
				mcp.WithString("new_content", mcp.Required(), mcp.Description("Replacement text (may be empty to delete)")),
				mcp.WithBoolean("replace_all", mcp.Description("Replace every occurrence (default: only if unique)")),
			),
			Handler: tools.EditHandler,
		},

		server.ServerTool{
			Tool: mcp.NewTool("glob",
				mcp.WithDescription("Find files by glob pattern (supports *, ?, **, [abc], {a,b})"),
				mcp.WithString("pattern", mcp.Required(), mcp.Description("Glob pattern, e.g. **/*.go or src/*.{ts,tsx}")),
				mcp.WithString("path", mcp.Description("Directory to search in (default: current directory)")),
			),
			Handler: tools.GlobHandler,
		},

		server.ServerTool{
			Tool: mcp.NewTool("grep",
				mcp.WithDescription("Search file contents with a regex (Go RE2 syntax)"),
				mcp.WithString("pattern", mcp.Required(), mcp.Description("Regular expression to search for")),
				mcp.WithString("path", mcp.Description("File or directory to search (default: current directory)")),
				mcp.WithString("include", mcp.Description("Glob filter for files, e.g. *.go or src/**/*.ts")),
				mcp.WithNumber("context", mcp.Description("Lines of context before/after each match (0-10)")),
				mcp.WithBoolean("ignore_case", mcp.Description("Case-insensitive search")),
			),
			Handler: tools.GrepHandler,
		},

		server.ServerTool{
			Tool: mcp.NewTool("multi_edit",
				mcp.WithDescription("Apply several exact-text replacements, possibly across files. Edits run in order; edits to the same file build on each other. If any edit fails validation, nothing is written."),
				mcp.WithArray("edits", mcp.Required(),
					mcp.Description("Ordered list of edits"),
					mcp.Items(map[string]any{
						"type": "object",
						"properties": map[string]any{
							"file_path":   map[string]any{"type": "string", "description": "Path to the file"},
							"old_content": map[string]any{"type": "string", "description": "Exact text to replace"},
							"new_content": map[string]any{"type": "string", "description": "Replacement text (may be empty to delete)"},
							"replace_all": map[string]any{"type": "boolean", "description": "Replace every occurrence (default: must be unique)"},
						},
						"required": []string{"file_path", "old_content", "new_content"},
					}),
				),
			),
			Handler: tools.MultiEditHandler,
		},

		server.ServerTool{
			Tool: mcp.NewTool("read",
				mcp.WithDescription("Read a text file. Each line is prefixed with [n]; do NOT include that prefix in edit old_content. Large files are paged with offset/limit."),
				mcp.WithString("file_path", mcp.Required(), mcp.Description("Path to the file")),
				mcp.WithNumber("offset", mcp.Description("Line to start from, 1-based (default 1)")),
				mcp.WithNumber("limit", mcp.Description("Max lines to return (default 2000, max 5000)")),
			),
			Handler: tools.ReadHandler,
		},

		server.ServerTool{
			Tool: mcp.NewTool("write",
				mcp.WithDescription("Create a NEW file (parent directories are created). Fails if the file exists; use edit to modify existing files."),
				mcp.WithString("file_path", mcp.Required(), mcp.Description("Path of the file to create")),
				mcp.WithString("content", mcp.Required(), mcp.Description("Full file content (may be empty)")),
			),
			Handler: tools.WriteHandler,
		},
	)

	return server.ServeStdio(s)
}
