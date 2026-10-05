package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type MultiEditResult struct {
	Success         bool         `json:"success"`
	Message         string       `json:"message"`
	Applied         int          `json:"applied"`
	Operations      []EditResult `json:"operations"`
	RolledBack      bool         `json:"rolled_back"`
	RollbackMessage string       `json:"rollback_message,omitempty"`
}

func MultiEdit(operations []EditResult) (*MultiEditResult, error) {
	result := &MultiEditResult{
		Operations: make([]EditResult, 0),
	}

	if len(operations) == 0 {
		return nil, fmt.Errorf("no operations provided")
	}

	// Ask user permission
	var opDetails strings.Builder
	opDetails.WriteString(fmt.Sprintf("Total operations: %d\n", len(operations)))
	for i, op := range operations {
		oldPreview := op.OldContent
		if len(oldPreview) > 40 {
			oldPreview = oldPreview[:40] + "..."
		}
		newPreview := op.NewContent
		if len(newPreview) > 40 {
			newPreview = newPreview[:40] + "..."
		}
		opDetails.WriteString(fmt.Sprintf("  [%d] File: %s\n      Replace: %q\n      With:    %q\n",
			i+1, op.Path, oldPreview, newPreview))
	}
	if err := AskPermission("multi_edit", opDetails.String()); err != nil {
		return nil, err
	}

	// ========== PHASE 1: VALIDATION ==========
	backups := make(map[string]string)
	validOps := make([]EditResult, 0)

	for i, op := range operations {
		if op.Path == "" {
			return nil, fmt.Errorf("op %d: path cannot be empty", i)
		}
		if op.OldContent == "" {
			return nil, fmt.Errorf("op %d: old_content cannot be empty", i)
		}
		if op.NewContent == "" {
			return nil, fmt.Errorf("op %d: new_content cannot be empty", i)
		}

		// --- Get raw content dari ReadFile (strip line number prefix) ---
		var readResult *ReadOutput
		var rerr error
		_ = SuppressPermission(func() error {
			readResult, rerr = ReadFile(op.Path)
			return rerr
		})
		if rerr != nil {
			return nil, fmt.Errorf("op %d (%s): %w", i, op.Path, rerr)
		}

		var rawContent string
		for _, line := range readResult.Lines {
			idx := strings.Index(line, "] ")
			if idx == -1 {
				return nil, fmt.Errorf("op %d (%s): unexpected line format: %s", i, op.Path, line)
			}
			rawContent += line[idx+2:] + "\n"
		}
		// --- end get raw content ---

		if !strings.Contains(rawContent, op.OldContent) {
			return nil, fmt.Errorf("op %d (%s): pattern not found", i, op.Path)
		}

		occurrences := strings.Count(rawContent, op.OldContent)

		newContent := strings.Replace(rawContent, op.OldContent, op.NewContent, 1)
		if newContent == rawContent {
			return nil, fmt.Errorf("op %d (%s): content unchanged after replacement", i, op.Path)
		}

		backups[op.Path] = rawContent

		validOps = append(validOps, EditResult{
			Path:        op.Path,
			OldContent:  op.OldContent,
			NewContent:  op.NewContent,
			Occurrences: occurrences,
		})
	}

	// ========== PHASE 2: EXECUTION ==========
	appliedPaths := []string{}
	var executionErr error

	for i, op := range validOps {
		originalContent := backups[op.Path]
		newContent := strings.Replace(originalContent, op.OldContent, op.NewContent, 1)

		// --- Atomic write (sama persis konsep EditFile) ---
		dir := filepath.Dir(op.Path)

		tmp, err := os.CreateTemp(dir, ".tmp-"+filepath.Base(op.Path)+"-*")
		if err != nil {
			executionErr = fmt.Errorf("op %d (%s): %w", i, op.Path, err)
			break
		}
		tmpName := tmp.Name()

		contentBytes := []byte(newContent)
		bytesWritten, werr := tmp.Write(contentBytes)
		if werr != nil {
			tmp.Close()
			os.Remove(tmpName)
			executionErr = fmt.Errorf("op %d (%s): %w", i, op.Path, werr)
			break
		}

		if bytesWritten != len(contentBytes) {
			tmp.Close()
			os.Remove(tmpName)
			executionErr = fmt.Errorf("op %d (%s): partial write: %d/%d bytes", i, op.Path, bytesWritten, len(contentBytes))
			break
		}

		if err := tmp.Sync(); err != nil {
			tmp.Close()
			os.Remove(tmpName)
			executionErr = fmt.Errorf("op %d (%s): %w", i, op.Path, err)
			break
		}

		if err := tmp.Chmod(0644); err != nil {
			tmp.Close()
			os.Remove(tmpName)
			executionErr = fmt.Errorf("op %d (%s): %w", i, op.Path, err)
			break
		}

		if err := tmp.Close(); err != nil {
			os.Remove(tmpName)
			executionErr = fmt.Errorf("op %d (%s): %w", i, op.Path, err)
			break
		}

		if err := os.Rename(tmpName, op.Path); err != nil {
			os.Remove(tmpName)
			executionErr = fmt.Errorf("op %d (%s): %w", i, op.Path, err)
			break
		}

		verifyData, err := os.ReadFile(op.Path)
		if err != nil {
			executionErr = fmt.Errorf("op %d (%s): %w", i, op.Path, err)
			break
		}
		if string(verifyData) != newContent {
			os.Remove(op.Path)
			executionErr = fmt.Errorf("op %d (%s): data integrity check failed after write", i, op.Path)
			break
		}
		// --- end atomic write ---

		appliedPaths = append(appliedPaths, op.Path)

		result.Operations = append(result.Operations, EditResult{
			Path:        op.Path,
			OldContent:  op.OldContent,
			NewContent:  op.NewContent,
			Status:      "success",
			Occurrences: op.Occurrences,
		})
	}

	// ========== PHASE 3: ROLLBACK (jika ada error) ==========
	if executionErr != nil {
		result.Success = false
		result.Message = executionErr.Error()
		result.RolledBack = true

		var rollbackErrs []error

		for _, path := range appliedPaths {
			originalContent := backups[path]

			// --- Atomic write untuk restore (konsep sama) ---
			dir := filepath.Dir(path)

			tmp, err := os.CreateTemp(dir, ".tmp-"+filepath.Base(path)+"-*")
			if err != nil {
				rollbackErrs = append(rollbackErrs, fmt.Errorf("rollback %s: %w", path, err))
				continue
			}
			tmpName := tmp.Name()

			contentBytes := []byte(originalContent)
			bytesWritten, werr := tmp.Write(contentBytes)
			if werr != nil {
				tmp.Close()
				os.Remove(tmpName)
				rollbackErrs = append(rollbackErrs, fmt.Errorf("rollback %s: %w", path, werr))
				continue
			}

			if bytesWritten != len(contentBytes) {
				tmp.Close()
				os.Remove(tmpName)
				rollbackErrs = append(rollbackErrs, fmt.Errorf("rollback %s: partial write", path))
				continue
			}

			if err := tmp.Sync(); err != nil {
				tmp.Close()
				os.Remove(tmpName)
				rollbackErrs = append(rollbackErrs, fmt.Errorf("rollback %s: %w", path, err))
				continue
			}

			if err := tmp.Chmod(0644); err != nil {
				tmp.Close()
				os.Remove(tmpName)
				rollbackErrs = append(rollbackErrs, fmt.Errorf("rollback %s: %w", path, err))
				continue
			}

			if err := tmp.Close(); err != nil {
				os.Remove(tmpName)
				rollbackErrs = append(rollbackErrs, fmt.Errorf("rollback %s: %w", path, err))
				continue
			}

			if err := os.Rename(tmpName, path); err != nil {
				os.Remove(tmpName)
				rollbackErrs = append(rollbackErrs, fmt.Errorf("rollback %s: %w", path, err))
				continue
			}
			// --- end atomic write restore ---
		}

		if len(rollbackErrs) > 0 {
			result.RollbackMessage = fmt.Sprintf("rollback had %d errors: %v", len(rollbackErrs), rollbackErrs)
			return result, executionErr
		}

		result.RollbackMessage = fmt.Sprintf("All %d changes rolled back", len(appliedPaths))
		return result, executionErr
	}

	// ========== SUCCESS ==========
	result.Success = true
	result.Applied = len(appliedPaths)
	result.Message = fmt.Sprintf("Successfully applied %d edits atomically", result.Applied)

	return result, nil
}
