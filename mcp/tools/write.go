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

const maxWriteSize = 10 << 20 // 10 MB

type WriteOutput struct {
	Path           string `json:"path"`
	Message        string `json:"message"`
	Size           int64  `json:"size"`
	CreatedParents bool   `json:"created_parents,omitempty"`
}

// WriteFile membuat file BARU. Gagal kalau file sudah ada (pakai edit untuk mengubahnya).
func WriteFile(filePath, content string) (*WriteOutput, error) {
	if strings.TrimSpace(filePath) == "" {
		return nil, fmt.Errorf("file_path cannot be empty")
	}
	if len(content) > maxWriteSize {
		return nil, fmt.Errorf("content too large (%d bytes, max %d)", len(content), maxWriteSize)
	}

	// Lstat: dangling symlink juga dianggap "sudah ada"
	if _, err := os.Lstat(filePath); err == nil {
		return nil, fmt.Errorf("file already exists: %s (use edit to modify it)", filePath)
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	dir := filepath.Dir(filePath)
	needMkdir := false
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		needMkdir = true
	}

	// Permission SEBELUM ada perubahan apa pun di disk
	desc := fmt.Sprintf("Target File: %s\nTotal Size: %d bytes", filePath, len(content))
	if needMkdir {
		desc += fmt.Sprintf("\nWill create parent directory: %s", dir)
	}
	desc += "\n--- Content Preview ---\n" + truncateRunes(content, 500)
	if err := AskPermission("write", desc); err != nil {
		return nil, err
	}

	if needMkdir {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	if err := writeFileAtomic(filePath, []byte(content), 0o644); err != nil {
		return nil, err
	}

	return &WriteOutput{
		Path:           filePath,
		Message:        "success",
		Size:           int64(len(content)),
		CreatedParents: needMkdir,
	}, nil
}

// WriteHandler adapts WriteFile to server.ToolHandlerFunc.
func WriteHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("file_path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	content, err := req.RequireString("content") // string kosong tetap valid
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	res, err := WriteFile(path, content)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	b, _ := json.Marshal(res)
	return mcp.NewToolResultText(string(b)), nil
}
