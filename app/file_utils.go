package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RemoveFile deletes a file at the given path.
// Returns an error if the file cannot be deleted.
func RemoveFile(path string) error {
	if path == "" {
		return fmt.Errorf("file path required")
	}

	// Check if file exists first (optional)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", path)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete file %s: %w", path, err)
	}

	return nil
}

// DeleteFilesWithPrefix deletes all files in a directory that start with the given prefix.
// Returns the number of deleted files and an error (if any).
func DeleteFilesWithPrefix(dir string, prefix string) (int, error) {
	if len(prefix) < 1 {
		return 0, fmt.Errorf("prefix required")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("failed to read directory: %w", err)
	}
	deletedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, prefix) {
			fullPath := filepath.Join(dir, name)
			if err := os.Remove(fullPath); err != nil {
				return deletedCount, fmt.Errorf("failed to delete %s: %w", fullPath, err)
			}
			deletedCount++
		}
	}
	return deletedCount, nil
}
