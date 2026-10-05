package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

type EditOp struct {
	FilePath   string `json:"file_path"`
	OldContent string `json:"old_content"`
	NewContent string `json:"new_content"`
	ReplaceAll bool   `json:"replace_all"`
}

type MultiEditOpResult struct {
	Index       int    `json:"index"` // 1-based
	Path        string `json:"path"`
	Occurrences int    `json:"occurrences"`
	Replaced    int    `json:"replaced"`
}

type MultiEditResult struct {
	Success         bool                `json:"success"`
	Message         string              `json:"message"`
	Applied         int                 `json:"applied"`
	FilesChanged    int                 `json:"files_changed"`
	Operations      []MultiEditOpResult `json:"operations,omitempty"`
	RolledBack      bool                `json:"rolled_back"`
	RollbackMessage string              `json:"rollback_message,omitempty"`
}

type fileEdit struct {
	realPath string
	perm     os.FileMode
	original string
	updated  string
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}

// MultiEdit menerapkan beberapa edit. Edit pada file yang sama diterapkan
// berurutan (edit ke-2 melihat hasil edit ke-1) dan file ditulis sekali.
// Semua divalidasi di memori dulu; kalau penulisan gagal, file yang sudah
// tertulis dikembalikan ke isi aslinya (best-effort).
func MultiEdit(ops []EditOp) (*MultiEditResult, error) {
	if len(ops) == 0 {
		return nil, fmt.Errorf("no edits provided")
	}

	// ===== PHASE 1: validasi + hitung hasil di memori =====
	files := map[string]*fileEdit{}
	var order []string
	opResults := make([]MultiEditOpResult, 0, len(ops))

	for i, op := range ops {
		n := i + 1
		if op.FilePath == "" {
			return nil, fmt.Errorf("edit %d: file_path cannot be empty", n)
		}
		if op.OldContent == "" {
			return nil, fmt.Errorf("edit %d: old_content cannot be empty", n)
		}
		if op.OldContent == op.NewContent {
			return nil, fmt.Errorf("edit %d: old_content and new_content are identical", n)
		}

		realPath, err := filepath.EvalSymlinks(op.FilePath)
		if err != nil {
			return nil, fmt.Errorf("edit %d (%s): %w", n, op.FilePath, err)
		}

		fe, ok := files[realPath]
		if !ok {
			info, err := os.Stat(realPath)
			if err != nil {
				return nil, fmt.Errorf("edit %d (%s): %w", n, op.FilePath, err)
			}
			if info.IsDir() {
				return nil, fmt.Errorf("edit %d: %s is a directory", n, op.FilePath)
			}
			data, err := os.ReadFile(realPath)
			if err != nil {
				return nil, fmt.Errorf("edit %d (%s): %w", n, op.FilePath, err)
			}
			fe = &fileEdit{
				realPath: realPath,
				perm:     info.Mode().Perm(),
				original: string(data),
				updated:  string(data),
			}
			files[realPath] = fe
			order = append(order, realPath)
		}

		occ := strings.Count(fe.updated, op.OldContent)
		switch {
		case occ == 0:
			return nil, fmt.Errorf("edit %d (%s): old content not found (edits earlier in this batch are already applied)", n, op.FilePath)
		case occ > 1 && !op.ReplaceAll:
			return nil, fmt.Errorf("edit %d (%s): old content found %d times; add more context or set replace_all", n, op.FilePath, occ)
		}

		count, replaced := 1, 1
		if op.ReplaceAll {
			count, replaced = -1, occ
		}
		fe.updated = strings.Replace(fe.updated, op.OldContent, op.NewContent, count)

		opResults = append(opResults, MultiEditOpResult{
			Index: n, Path: op.FilePath, Occurrences: occ, Replaced: replaced,
		})
	}

	// ===== PHASE 2: minta permission (setelah semua valid) =====
	var sb strings.Builder
	fmt.Fprintf(&sb, "%d edit(s) across %d file(s)\n", len(ops), len(order))
	for i, op := range ops {
		fmt.Fprintf(&sb, "  [%d] %s\n      - %q\n      + %q\n",
			i+1, op.FilePath, truncateRunes(op.OldContent, 60), truncateRunes(op.NewContent, 60))
	}
	if err := AskPermission("multi_edit", sb.String()); err != nil {
		return nil, err
	}

	// ===== PHASE 3: tulis per file, rollback kalau gagal =====
	var written []*fileEdit
	for _, p := range order {
		fe := files[p]
		if fe.updated == fe.original { // mis. A->B lalu B->A
			continue
		}
		if err := writeFileAtomic(fe.realPath, []byte(fe.updated), fe.perm); err != nil {
			execErr := fmt.Errorf("write %s: %w", fe.realPath, err)
			res := &MultiEditResult{Message: execErr.Error()}

			// File yang gagal tidak berubah (rename atomic), cukup restore yang sudah tertulis.
			var rbErrs []string
			for _, w := range written {
				if rerr := writeFileAtomic(w.realPath, []byte(w.original), w.perm); rerr != nil {
					rbErrs = append(rbErrs, fmt.Sprintf("%s: %v", w.realPath, rerr))
				}
			}
			res.RolledBack = len(written) > 0 && len(rbErrs) == 0
			switch {
			case len(rbErrs) > 0:
				res.RollbackMessage = fmt.Sprintf("rollback failed for %d file(s): %s", len(rbErrs), strings.Join(rbErrs, "; "))
			case len(written) > 0:
				res.RollbackMessage = fmt.Sprintf("%d file(s) restored to original", len(written))
			default:
				res.RollbackMessage = "no files had been modified"
			}
			return res, execErr
		}
		written = append(written, fe)
	}

	return &MultiEditResult{
		Success:      true,
		Message:      fmt.Sprintf("applied %d edit(s) to %d file(s)", len(ops), len(written)),
		Applied:      len(ops),
		FilesChanged: len(written),
		Operations:   opResults,
	}, nil
}

// MultiEditHandler adapts MultiEdit to server.ToolHandlerFunc.
func MultiEditHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Edits []EditOp `json:"edits"`
	}
	if err := req.BindArguments(&args); err != nil {
		return mcp.NewToolResultError("invalid arguments: " + err.Error()), nil
	}

	res, err := MultiEdit(args.Edits)
	if err != nil {
		if res != nil { // gagal di tengah penulisan: kirim detail rollback
			b, _ := json.Marshal(res)
			return mcp.NewToolResultError(string(b)), nil
		}
		return mcp.NewToolResultError(err.Error()), nil
	}

	b, _ := json.Marshal(res)
	return mcp.NewToolResultText(string(b)), nil
}
