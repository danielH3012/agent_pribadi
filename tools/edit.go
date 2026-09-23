package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type EditResult struct {
	OldContent  string `json:"old_content"`
	NewContent  string `json:"new_content"`
	Status      string `json:"status"`
	Occurrences int    `json:"occurrences,omitempty"`
}

func EditFile(filePath string, newContent string, oldContent string) (*EditResult, error) {
	if oldContent == "" {
		return nil, fmt.Errorf("old_content cannot be empty")
	}
	if newContent == "" {
		return nil, fmt.Errorf("new_content cannot be empty")
	}

	readResult, err := ReadFile(filePath) // Get formatted dengan line numbers
	if err != nil {
		return nil, err
	}

	var rawContent string
	for _, line := range readResult.Lines {
		i := strings.Index(line, "] ")
		if i == -1 {
			return nil, fmt.Errorf("unexpected line format: %s", line)
		}
		line = line[i+2:] // Remove line number prefix
		rawContent += line + "\n"
	}

	// 4. Count occurrences
	occurrences := strings.Count(rawContent, oldContent)

	if strings.Contains(rawContent, oldContent) {
		// 5. Replace (only first occurrence)
		updatedContent := strings.Replace(rawContent, oldContent, newContent, 1)

		// 6. Verify content changed
		if updatedContent == rawContent {
			return nil, fmt.Errorf("content unchanged after replacement")
		}

		dir := filepath.Dir(filePath)

		tmp, err := os.CreateTemp(dir, ".tmp-"+filepath.Base(filePath)+"-*")
		if err != nil {
			return nil, err
		}
		tmpName := tmp.Name()

		// Clean up on any failure
		defer func() {
			if tmpName != "" {
				_ = tmp.Close()
				_ = os.Remove(tmpName)
			}
		}()

		// ✅ 5. Write data + VERIFY bytes written
		contentBytes := []byte(updatedContent)
		bytesWritten, err := tmp.Write(contentBytes)
		if err != nil {
			return nil, err
		}

		// ✅ Verify semua bytes tertulis
		if bytesWritten != len(contentBytes) {
			return nil, fmt.Errorf("partial write: %d/%d bytes", bytesWritten, len(contentBytes))
		}

		// 3. fsync to ensure data hits disk before rename
		if err := tmp.Sync(); err != nil {
			return nil, err
		}

		// 3. Set permissions (0644 = rw-r--r--)
		if err := tmp.Chmod(0644); err != nil {
			return nil, err
		}

		if err := tmp.Close(); err != nil {
			return nil, err
		}

		if err := os.Rename(tmpName, filePath); err != nil {
			return nil, err
		}
		tmpName = ""

		verifyData, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		if string(verifyData) != updatedContent {
			// Data tidak match - ini serious error
			os.Remove(filePath) // Clean up corrupted file
			return nil, fmt.Errorf("data integrity check failed after write")
		}
		return &EditResult{Status: "success", OldContent: oldContent, NewContent: newContent, Occurrences: occurrences}, nil
	} else {
		return nil, fmt.Errorf("old content not found in file")
	}
}
