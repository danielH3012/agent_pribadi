package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

const (
	maxGrepMatches  = 500
	maxGrepFileSize = 5 << 20 // 5 MB
	maxLineLen      = 500
	maxContext      = 10
)

type GrepMatch struct {
	File   string   `json:"file"`
	Line   int      `json:"line"`
	Text   string   `json:"text"`
	Before []string `json:"before,omitempty"`
	After  []string `json:"after,omitempty"`
}

type GrepOptions struct {
	Include    string // glob filter, e.g. "*.go" atau "src/**/*.ts"
	Context    int    // jumlah baris sebelum/sesudah match
	IgnoreCase bool
}

type GrepResult struct {
	Pattern       string      `json:"pattern"`
	Root          string      `json:"root"`
	Matches       []GrepMatch `json:"matches"`
	Count         int         `json:"count"`
	FilesSearched int         `json:"files_searched"`
	Truncated     bool        `json:"truncated"`
}

func clipLine(s string) string {
	r := []rune(s)
	if len(r) <= maxLineLen {
		return s
	}
	return string(r[:maxLineLen]) + "…"
}

// Grep mencari regex di file di bawah root (atau satu file kalau root adalah file).
func Grep(ctx context.Context, root, pattern string, opts GrepOptions) (*GrepResult, error) {
	if strings.TrimSpace(pattern) == "" {
		return nil, fmt.Errorf("pattern cannot be empty")
	}
	if root == "" {
		root = "."
	}
	if opts.Context < 0 {
		opts.Context = 0
	}
	if opts.Context > maxContext {
		opts.Context = maxContext
	}

	expr := pattern
	if opts.IgnoreCase {
		expr = "(?i)" + expr
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %w", err)
	}

	var includeRe *regexp.Regexp
	includeByName := false
	if opts.Include != "" {
		includeRe, err = GlobToRegex(opts.Include)
		if err != nil {
			return nil, fmt.Errorf("invalid include glob: %w", err)
		}
		includeByName = !strings.Contains(opts.Include, "/") // "*.go" cocokkan nama file saja
	}

	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}

	if err := AskPermission("grep", fmt.Sprintf("Search regex %q in %q (include: %q, context: %d, ignore_case: %v)",
		pattern, root, opts.Include, opts.Context, opts.IgnoreCase)); err != nil {
		return nil, err
	}

	res := &GrepResult{Pattern: pattern, Root: root, Matches: []GrepMatch{}}

	// searchFile mengembalikan true kalau pencarian harus berhenti (limit tercapai).
	searchFile := func(path, display string, size int64) bool {
		if size > maxGrepFileSize {
			return false
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		head := data
		if len(head) > 8000 {
			head = head[:8000]
		}
		if bytes.IndexByte(head, 0) >= 0 { // binary
			return false
		}
		res.FilesSearched++

		lines := strings.Split(string(data), "\n")
		if n := len(lines); n > 0 && lines[n-1] == "" {
			lines = lines[:n-1] // buang elemen kosong akibat trailing newline
		}

		for i, line := range lines {
			line = strings.TrimSuffix(line, "\r")
			if !re.MatchString(line) {
				continue
			}
			if len(res.Matches) >= maxGrepMatches {
				res.Truncated = true
				return true
			}

			m := GrepMatch{File: display, Line: i + 1, Text: clipLine(strings.TrimSpace(line))}
			if opts.Context > 0 {
				start := i - opts.Context
				if start < 0 {
					start = 0
				}
				end := i + opts.Context + 1
				if end > len(lines) {
					end = len(lines)
				}
				for _, l := range lines[start:i] {
					m.Before = append(m.Before, clipLine(strings.TrimSuffix(l, "\r")))
				}
				for _, l := range lines[i+1 : end] {
					m.After = append(m.After, clipLine(strings.TrimSuffix(l, "\r")))
				}
			}
			res.Matches = append(res.Matches, m)
		}
		return false
	}

	if !info.IsDir() {
		searchFile(root, root, info.Size())
		res.Count = len(res.Matches)
		return res, nil
	}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if cerr := ctx.Err(); cerr != nil {
			return cerr
		}
		if err != nil { // mis. permission denied: lanjut
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if path != root && skipDirs[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() { // skip symlink, socket, dll.
			return nil
		}

		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)

		if includeRe != nil {
			target := rel
			if includeByName {
				target = d.Name()
			}
			if !includeRe.MatchString(target) {
				return nil
			}
		}

		fi, ierr := d.Info()
		if ierr != nil {
			return nil
		}
		if searchFile(path, rel, fi.Size()) {
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	res.Count = len(res.Matches)
	return res, nil
}

// GrepHandler adapts Grep to server.ToolHandlerFunc.
func GrepHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pattern, err := req.RequireString("pattern")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	opts := GrepOptions{
		Include:    req.GetString("include", ""),
		Context:    req.GetInt("context", 0),
		IgnoreCase: req.GetBool("ignore_case", false),
	}

	res, err := Grep(ctx, req.GetString("path", "."), pattern, opts)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	b, _ := json.Marshal(res)
	return mcp.NewToolResultText(string(b)), nil
}
