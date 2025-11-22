package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TempDirManager manages temporary directories for tests
type TempDirManager struct {
	baseDir string
	dirs    []string
}

// NewTempDirManager creates a new temporary directory manager
func NewTempDirManager(prefix string) (*TempDirManager, error) {
	baseDir, err := os.MkdirTemp("", prefix)
	if err != nil {
		return nil, fmt.Errorf("creating base temp dir: %w", err)
	}

	return &TempDirManager{
		baseDir: baseDir,
		dirs:    []string{baseDir},
	}, nil
}

// NewTempDirManagerForExisting creates a new TempDirManager for a path that
// already exists. The manager will not delete this path on Cleanup since it
// was created by an external process (the orchestrator).
func NewTempDirManagerForExisting(path string) (*TempDirManager, error) {
	// Verify the path exists
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("existing path does not exist: %w", err)
	}

	return &TempDirManager{
		baseDir: path,
		dirs:    []string{}, // Empty dirs list means Cleanup won't remove anything
	}, nil
}

// BaseDir returns the root temporary directory
func (m *TempDirManager) BaseDir() string {
	return m.baseDir
}

// CreateDir creates a subdirectory within the temp space
func (m *TempDirManager) CreateDir(name string) (string, error) {
	// Sanitize name to prevent directory traversal
	name = strings.ReplaceAll(name, "..", "")
	name = strings.ReplaceAll(name, "/", "-")

	path := filepath.Join(m.baseDir, name)
	if err := CreateDir(path); err != nil {
		return "", err
	}

	m.dirs = append(m.dirs, path)
	return path, nil
}

// Cleanup removes all created directories
func (m *TempDirManager) Cleanup() error {
	// Remove in reverse order to handle nested directories
	for i := len(m.dirs) - 1; i >= 0; i-- {
		if err := os.RemoveAll(m.dirs[i]); err != nil {
			return fmt.Errorf("removing %s: %w", m.dirs[i], err)
		}
	}
	return nil
}