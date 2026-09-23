package tools

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type GrepMatch struct {
	File   string   `json:"file"`
	Line   int      `json:"line"`
	Text   string   `json:"text"`
	Before []string `json:"before,omitempty"` // Context before
	After  []string `json:"after,omitempty"`  // Context after
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Grep - Simple pattern search
func Grep(pattern string, searchPath string) ([]GrepMatch, error) {
	// Ask user permission
	if err := AskPermission("grep", fmt.Sprintf("Search regex '%s' in directory '%s'", pattern, searchPath)); err != nil {
		return nil, err
	}

	// 1. Compile regex
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %w", err)
	}

	var results []GrepMatch

	// 2. Walk directory
	filepath.WalkDir(searchPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		// 3. Read file
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip unreadable files
		}

		// 4. Search lines
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if regex.MatchString(line) {
				// Match found
				match := GrepMatch{
					File: path,
					Line: i + 1,
					Text: strings.TrimSpace(line),
				}
				results = append(results, match)
			}
		}

		return nil
	})

	return results, nil
}

// GrepWithContext - Search with context lines
func GrepWithContext(pattern string, searchPath string, contextLines int) ([]GrepMatch, error) {
	// Ask user permission
	if err := AskPermission("grep", fmt.Sprintf("Search regex '%s' in directory '%s' (context: %d lines)", pattern, searchPath, contextLines)); err != nil {
		return nil, err
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %w", err)
	}

	var results []GrepMatch

	filepath.WalkDir(searchPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if regex.MatchString(line) {
				match := GrepMatch{
					File: path,
					Line: i + 1,
					Text: strings.TrimSpace(line),
				}

				// Add context (before + after)
				start := max(0, i-contextLines)
				end := min(len(lines), i+contextLines+1)

				if start < i {
					match.Before = lines[start:i]
				}
				if i+1 < end {
					match.After = lines[i+1 : end]
				}

				results = append(results, match)
			}
		}

		return nil
	})

	return results, nil
}
