package tools

import (
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

const maxGlobResults = 1000

// Direktori yang di-skip saat walk. Hapus entri kalau perlu ikut di-scan.
var skipDirs = map[string]bool{".git": true, "node_modules": true}

type GlobResult struct {
	Pattern   string   `json:"pattern"`
	Root      string   `json:"root"`
	Matches   []string `json:"matches"`
	Count     int      `json:"count"`
	Truncated bool     `json:"truncated"`
}

// GlobToRegex converts a glob pattern to an anchored regexp.
// Supports: * ? ** [abc] [!abc] {a,b}
func GlobToRegex(pattern string) (*regexp.Regexp, error) {
	pattern = filepath.ToSlash(pattern)

	var sb strings.Builder
	sb.WriteString("^")
	inBrace := false

	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch c {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				i++ // konsumsi '*' kedua
				if i+1 < len(pattern) && pattern[i+1] == '/' {
					i++                        // konsumsi '/'
					sb.WriteString("(?:.*/)?") // "**/" = nol atau lebih direktori
				} else {
					sb.WriteString(".*")
				}
			} else {
				sb.WriteString("[^/]*")
			}
		case '?':
			sb.WriteString("[^/]")
		case '[':
			j := i + 1
			if j < len(pattern) && (pattern[j] == '!' || pattern[j] == '^') {
				j++
			}
			if j < len(pattern) && pattern[j] == ']' {
				j++
			}
			for j < len(pattern) && pattern[j] != ']' {
				j++
			}
			if j >= len(pattern) { // tidak ada penutup -> literal
				sb.WriteString(`\[`)
				break
			}
			class := pattern[i+1 : j]
			if class[0] == '!' {
				class = "^" + class[1:]
			}
			sb.WriteString("[" + class + "]")
			i = j
		case '{':
			inBrace = true
			sb.WriteString("(?:")
		case '}':
			if inBrace {
				inBrace = false
				sb.WriteString(")")
			} else {
				sb.WriteString(`\}`)
			}
		case ',':
			if inBrace {
				sb.WriteString("|")
			} else {
				sb.WriteString(",")
			}
		case '.', '+', '^', '$', '(', ')', '|', '\\', ']':
			sb.WriteByte('\\')
			sb.WriteByte(c)
		default:
			sb.WriteByte(c)
		}
	}
	sb.WriteString("$")

	return regexp.Compile(sb.String())
}

// Glob finds files under root matching pattern (paths returned relative to root).
func Glob(ctx context.Context, root, pattern string) (*GlobResult, error) {
	pattern = strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(pattern)), "./")
	if pattern == "" {
		return nil, fmt.Errorf("pattern cannot be empty")
	}
	if root == "" {
		root = "."
	}

	re, err := GlobToRegex(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern: %w", err)
	}

	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}

	recursive := strings.Contains(pattern, "**")
	maxDepth := strings.Count(pattern, "/") + 1 // jumlah segmen pattern

	if err := AskPermission("glob", fmt.Sprintf("Search %q in %s (recursive: %v)", pattern, root, recursive)); err != nil {
		return nil, err
	}

	res := &GlobResult{Pattern: pattern, Root: root, Matches: []string{}}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if cerr := ctx.Err(); cerr != nil {
			return cerr
		}
		if err != nil { // mis. permission denied: lanjut, jangan batalkan seluruh walk
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		rel, rerr := filepath.Rel(root, path)
		if rerr != nil || rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			if skipDirs[d.Name()] {
				return fs.SkipDir
			}
			// tanpa "**", file di dalam dir pada kedalaman >= maxDepth tidak mungkin match
			if !recursive && strings.Count(rel, "/")+1 >= maxDepth {
				return fs.SkipDir
			}
			return nil
		}

		if re.MatchString(rel) {
			if len(res.Matches) >= maxGlobResults {
				res.Truncated = true
				return fs.SkipAll
			}
			res.Matches = append(res.Matches, rel) // WalkDir sudah urut leksikal
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	res.Count = len(res.Matches)
	return res, nil
}

// GlobHandler adapts Glob to server.ToolHandlerFunc.
func GlobHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pattern, err := req.RequireString("pattern")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	root := req.GetString("path", ".")

	res, err := Glob(ctx, root, pattern)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	b, _ := json.Marshal(res)
	return mcp.NewToolResultText(string(b)), nil
}
