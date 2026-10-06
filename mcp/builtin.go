package mcp

import (
	"context"
	"fmt"

	"agentPribadi/mcp/tools"
)

type BashInput struct {
	Command string `json:"command" desc:"The shell command to execute"`
}

type EditInput struct {
	FilePath   string `json:"file_path,omitempty" desc:"The path of the file to edit"`
	Path       string `json:"path,omitempty" desc:"Alias for file_path"`
	OldContent string `json:"old_content" desc:"The exact text to replace"`
	NewContent string `json:"new_content" desc:"The replacement text"`
	ReplaceAll bool   `json:"replace_all,omitempty" desc:"Replace every occurrence (default: only if unique)"`
}

type GlobInput struct {
	Pattern string `json:"pattern" desc:"Glob pattern to match files against, e.g. **/*.go"`
	Path    string `json:"path,omitempty" desc:"Directory to search in (default: current directory)"`
}

type GrepInput struct {
	Pattern      string `json:"pattern" desc:"Regular expression pattern to search for"`
	Path         string `json:"path,omitempty" desc:"Directory or file path to search within"`
	SearchPath   string `json:"search_path,omitempty" desc:"Alias for path"`
	Include      string `json:"include,omitempty" desc:"Glob filter for files, e.g. *.go"`
	Context      int    `json:"context,omitempty" desc:"Lines of context before/after each match"`
	ContextLines int    `json:"context_lines,omitempty" desc:"Alias for context"`
	IgnoreCase   bool   `json:"ignore_case,omitempty" desc:"Case-insensitive search"`
}

type MultiEditInput struct {
	Edits      []tools.EditOp `json:"edits,omitempty" desc:"List of edit operations to apply atomically"`
	Operations []tools.EditOp `json:"operations,omitempty" desc:"Alias for edits"`
}

type ReadInput struct {
	FilePath string `json:"file_path,omitempty" desc:"Path of the file to read"`
	Path     string `json:"path,omitempty" desc:"Alias for file_path"`
	Offset   int    `json:"offset,omitempty" desc:"Line to start from, 1-based (default 1)"`
	Limit    int    `json:"limit,omitempty" desc:"Max lines to return (default 2000)"`
}

type WriteInput struct {
	FilePath string `json:"file_path,omitempty" desc:"Path of the file to create and write to"`
	Path     string `json:"path,omitempty" desc:"Alias for file_path"`
	Content  string `json:"content" desc:"Full file content to write"`
}

// BuiltinTools returns the standard built-in tools (bash, edit, glob, grep, multi_edit, read, write)
func BuiltinTools() []Tool {
	return []Tool{
		Func("bash", "Execute a shell command and capture its stdout, exit code, and status.",
			func(ctx context.Context, in BashInput) (any, error) {
				return tools.Bash(ctx, in.Command)
			},
		),

		Func("edit", "Edit a file by replacing old content with new content atomically.",
			func(ctx context.Context, in EditInput) (any, error) {
				p := in.FilePath
				if p == "" {
					p = in.Path
				}
				if p == "" {
					return nil, fmt.Errorf("file_path cannot be empty")
				}
				return tools.EditFile(p, in.OldContent, in.NewContent, in.ReplaceAll)
			},
		),

		Func("glob", "Find files matching a glob pattern (supports *, **, ?, etc.).",
			func(ctx context.Context, in GlobInput) (any, error) {
				root := in.Path
				if root == "" {
					root = "."
				}
				return tools.Glob(ctx, root, in.Pattern)
			},
			ReadOnly(),
		),

		Func("grep", "Search file contents with a regular expression pattern.",
			func(ctx context.Context, in GrepInput) (any, error) {
				p := in.Path
				if p == "" {
					p = in.SearchPath
				}
				ctxLines := in.Context
				if ctxLines == 0 {
					ctxLines = in.ContextLines
				}
				opts := tools.GrepOptions{
					Include:    in.Include,
					Context:    ctxLines,
					IgnoreCase: in.IgnoreCase,
				}
				return tools.Grep(ctx, p, in.Pattern, opts)
			},
			ReadOnly(),
		),

		Func("multi_edit", "Apply several exact-text replacements atomically across files.",
			func(ctx context.Context, in MultiEditInput) (any, error) {
				edits := in.Edits
				if len(edits) == 0 {
					edits = in.Operations
				}
				return tools.MultiEdit(edits)
			},
		),

		Func("read", "Read contents of a text file with line numbers and pagination.",
			func(ctx context.Context, in ReadInput) (any, error) {
				p := in.FilePath
				if p == "" {
					p = in.Path
				}
				if p == "" {
					return nil, fmt.Errorf("file_path cannot be empty")
				}
				return tools.ReadFile(p, in.Offset, in.Limit)
			},
			ReadOnly(),
		),

		Func("write", "Create a NEW file and write content to it atomically.",
			func(ctx context.Context, in WriteInput) (any, error) {
				p := in.FilePath
				if p == "" {
					p = in.Path
				}
				if p == "" {
					return nil, fmt.Errorf("file_path cannot be empty")
				}
				return tools.WriteFile(p, in.Content)
			},
		),
	}
}

// DefaultServer creates a new Server pre-populated with all built-in tools.
func DefaultServer(opts ...Option) *Server {
	s := New("qtera-agent", "2.0.0", opts...)
	_ = s.Add(BuiltinTools()...)
	return s
}
