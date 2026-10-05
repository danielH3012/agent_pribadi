package tools

import (
	"fmt"
	"os"
	"strings"
)

type ReadOutput struct {
	Path    string   `json:"path"`
	Lines   []string `json:"lines"`
	Total   int      `json:"total"`
	Limited bool     `json:"limited,omitempty"`
	Limit   int      `json:"limit,omitempty"`
}

func ReadFile(path string, limit ...int) (*ReadOutput, error) {
	// Ask user permission
	desc := fmt.Sprintf("Target File: %s", path)
	if len(limit) > 0 && limit[0] > 0 {
		desc = fmt.Sprintf("Target File: %s (limit: %d lines)", path, limit[0])
	}
	if err := AskPermission("read", desc); err != nil {
		return nil, err
	}

	// 1. Validate path
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("file not found: %s", path)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory, not a file", path)
	}

	// 2. Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}

	// 3. Split by lines
	content := string(data)
	rawLines := strings.Split(content, "\n")

	// 4. Format dengan line numbers
	output := &ReadOutput{
		Path:  path,
		Total: len(rawLines),
	}

	// 5. Apply limit jika ada
	linesToShow := rawLines
	if len(limit) > 0 && limit[0] > 0 {
		maxLines := limit[0]
		if len(rawLines) > maxLines {
			linesToShow = rawLines[:maxLines]
			output.Limited = true
			output.Limit = maxLines
		}
	}

	// 6. Format output
	for i, line := range linesToShow {
		output.Lines = append(output.Lines, fmt.Sprintf("[%d] %s", i+1, line))
	}

	return output, nil
}
