package tools

import (
	"fmt"
	"os"
	"path/filepath"
)

type WriteOutput struct {
	Path    string `json:"path"`
	Message string `json:"message"`
	Size    int64  `json:"size"`
}

func WriteFile(filePath string, content string) (*WriteOutput, error) {
	// 1. Check file sudah ada
	_, err := os.Stat(filePath)
	if err == nil {
		return nil, fmt.Errorf("file already exists: %s", filePath)
	}
	if !os.IsNotExist(err) {
		return nil, err
	}

	// ✅ 2. Create parent directories jika belum ada
	dir := filepath.Dir(filePath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}
	// 3. Validate content
	if content == "" {
		return nil, fmt.Errorf("content cannot be empty")
	}

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
	contentBytes := []byte(content)
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

	// 5. Atomic rename
	if err := os.Rename(tmpName, filePath); err != nil {
		return nil, err
	}
	tmpName = "" // prevent deferred cleanup

	// ✅ VERIFY: Read back dan compare (integrity check)
	verifyData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	if string(verifyData) != content {
		// Data tidak match - ini serious error
		os.Remove(filePath) // Clean up corrupted file
		return nil, fmt.Errorf("data integrity check failed after write")
	}

	return &WriteOutput{Path: filePath, Message: "success", Size: int64(len(content))}, nil
}
