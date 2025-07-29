package fs

import (
	"os"
	"path/filepath"
)

// CleanPath returns the absolute path with symlinks resolved
func CleanPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

// EnsureDir ensures a directory exists, creating it if necessary
func EnsureDir(path string) error {
	if !IsDir(path) {
		return CreateDir(path)
	}
	return nil
}

// RemoveIfExists removes a file or directory if it exists
func RemoveIfExists(path string) error {
	if Exists(path) {
		return os.RemoveAll(path)
	}
	return nil
}

// ListFiles returns all files in a directory (non-recursive)
func ListFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

// ListDirs returns all subdirectories in a directory (non-recursive)
func ListDirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}

	return dirs, nil
}