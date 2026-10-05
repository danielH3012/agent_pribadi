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

type EditResult struct {
	Path        string `json:"path"`
	Status      string `json:"status"`
	Occurrences int    `json:"occurrences"`
	Replaced    int    `json:"replaced"`
	OldContent  string `json:"old_content"`
	NewContent  string `json:"new_content"`
}

func EditFile(filePath, oldContent, newContent string, replaceAll bool) (*EditResult, error) {
	if oldContent == "" {
		return nil, fmt.Errorf("old_content cannot be empty")
	}
	if oldContent == newContent {
		return nil, fmt.Errorf("old_content and new_content are identical")
	}

	// Resolve symlink supaya rename tidak menimpa symlink-nya
	realPath, err := filepath.EvalSymlinks(filePath)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(realPath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory", filePath)
	}

	data, err := os.ReadFile(realPath)
	if err != nil {
		return nil, err
	}
	raw := string(data)

	// Validasi dulu sebelum minta permission
	occurrences := strings.Count(raw, oldContent)
	switch {
	case occurrences == 0:
		return nil, fmt.Errorf("old content not found in file")
	case occurrences > 1 && !replaceAll:
		return nil, fmt.Errorf("old content found %d times; add more surrounding context or set replace_all", occurrences)
	}

	n, replaced := 1, 1
	if replaceAll {
		n, replaced = -1, occurrences
	}
	updated := strings.Replace(raw, oldContent, newContent, n)

	desc := fmt.Sprintf("Target File: %s (%d occurrence(s) will be replaced)\n--- Old Content (%d chars) ---\n%s\n--- New Content (%d chars) ---\n%s",
		filePath, replaced, len(oldContent), oldContent, len(newContent), newContent)
	if err := AskPermission("edit", desc); err != nil {
		return nil, err
	}

	if err := writeFileAtomic(realPath, []byte(updated), info.Mode().Perm()); err != nil {
		return nil, err
	}

	return &EditResult{
		Path:        filePath,
		Status:      "success",
		Occurrences: occurrences,
		Replaced:    replaced,
		OldContent:  oldContent,
		NewContent:  newContent,
	}, nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-"+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if tmpName != "" {
			_ = os.Remove(tmpName)
		}
	}()

	_, werr := tmp.Write(data) // Write sudah return error kalau partial write
	if werr == nil {
		werr = tmp.Chmod(perm) // pertahankan permission asli
	}
	if werr == nil {
		werr = tmp.Sync()
	}
	cerr := tmp.Close()
	if werr != nil {
		return werr
	}
	if cerr != nil {
		return cerr
	}

	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	tmpName = ""
	return nil
}

// EditHandler adapts EditFile to server.ToolHandlerFunc.
func EditHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("file_path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	oldContent, err := req.RequireString("old_content")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	newContent, err := req.RequireString("new_content")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	replaceAll := req.GetBool("replace_all", false)

	res, err := EditFile(path, oldContent, newContent, replaceAll)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	b, _ := json.Marshal(res)
	return mcp.NewToolResultText(string(b)), nil
}
