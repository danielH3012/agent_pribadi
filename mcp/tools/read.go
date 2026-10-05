package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

const (
	defaultReadLimit = 2000
	maxReadLimit     = 5000
	maxReadFileSize  = 10 << 20 // 10 MB
	maxReadLineLen   = 2000
)

type ReadOutput struct {
	Path       string   `json:"path"`
	Lines      []string `json:"lines"`
	Total      int      `json:"total"`
	Offset     int      `json:"offset"` // 1-based
	Truncated  bool     `json:"truncated"`
	NextOffset int      `json:"next_offset,omitempty"`
}

func clipRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// ReadFile membaca file teks dengan nomor baris. offset 1-based, limit <= 0 memakai default.
func ReadFile(path string, offset, limit int) (*ReadOutput, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("file_path cannot be empty")
	}
	if offset < 1 {
		offset = 1
	}
	if limit <= 0 {
		limit = defaultReadLimit
	}
	if limit > maxReadLimit {
		limit = maxReadLimit
	}

	// Validasi dulu sebelum minta permission
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory, not a file", path)
	}
	if info.Size() > maxReadFileSize {
		return nil, fmt.Errorf("file too large (%d bytes, max %d); use grep to search it instead", info.Size(), maxReadFileSize)
	}

	if err := AskPermission("read", fmt.Sprintf("Target File: %s (offset: %d, limit: %d lines)", path, offset, limit)); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}

	head := data
	if len(head) > 8000 {
		head = head[:8000]
	}
	if bytes.IndexByte(head, 0) >= 0 {
		return nil, fmt.Errorf("%s looks like a binary file", path)
	}

	lines := strings.Split(string(data), "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1] // trailing newline bukan baris tambahan
	}
	total := len(lines)

	out := &ReadOutput{Path: path, Total: total, Offset: offset, Lines: []string{}}
	if total == 0 {
		return out, nil
	}
	if offset > total {
		return nil, fmt.Errorf("offset %d is beyond end of file (%d lines)", offset, total)
	}

	end := offset - 1 + limit
	if end > total {
		end = total
	}
	for i := offset - 1; i < end; i++ {
		line := clipRunes(strings.TrimSuffix(lines[i], "\r"), maxReadLineLen)
		out.Lines = append(out.Lines, fmt.Sprintf("[%d] %s", i+1, line))
	}
	if end < total {
		out.Truncated = true
		out.NextOffset = end + 1
	}
	return out, nil
}

// ReadHandler adapts ReadFile to server.ToolHandlerFunc.
func ReadHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("file_path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	res, err := ReadFile(path, req.GetInt("offset", 1), req.GetInt("limit", 0))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if res.Total == 0 {
		return mcp.NewToolResultText("(empty file)"), nil
	}

	var sb strings.Builder
	sb.WriteString(strings.Join(res.Lines, "\n"))
	if res.Truncated {
		fmt.Fprintf(&sb, "\n\n[truncated: showing lines %d-%d of %d; call again with offset=%d]",
			res.Offset, res.NextOffset-1, res.Total, res.NextOffset)
	}
	return mcp.NewToolResultText(sb.String()), nil
}
