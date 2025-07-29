package fs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CreateDir creates a directory with all parent directories
func CreateDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// WriteFile writes content to a file, creating parent directories as needed
func WriteFile(path string, content []byte) error {
	dir := filepath.Dir(path)
	if err := CreateDir(dir); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}

	return os.WriteFile(path, content, 0644)
}

// WriteString writes a string to a file
func WriteString(path string, content string) error {
	return WriteFile(path, []byte(content))
}

// ReadString reads a file and returns its content as a string
func ReadString(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}
	return string(content), nil
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source file: %w", err)
	}
	defer sourceFile.Close()

	dir := filepath.Dir(dst)
	if err := CreateDir(dir); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("copying file content: %w", err)
	}

	return nil
}

// Exists checks if a path exists
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDir checks if a path is a directory
func IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// IsFile checks if a path is a regular file
func IsFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}