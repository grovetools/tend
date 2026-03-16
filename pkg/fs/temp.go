package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// CreateShortTempDir creates a temporary directory guaranteeing a short path
// (explicitly using /tmp on macOS/Linux) to avoid Unix socket length limits.
// Unix domain sockets have a maximum path length of ~104 characters on macOS
// and ~108 characters on Linux. The default TMPDIR on macOS (e.g.,
// /var/folders/4j/w6twdjd14r97l3n80z2t64n00000gn/T) consumes most of this
// budget, causing socket creation to fail for paths like groved.sock.
//
// This function forces temp directory creation in /tmp on Unix systems,
// ensuring socket paths remain short enough to be valid.
func CreateShortTempDir(prefix string) (string, error) {
	dir := ""
	// Force /tmp on macOS/Linux; let Windows use default
	if runtime.GOOS != "windows" {
		dir = "/tmp"
	}
	// MkdirTemp creates directories with 0700 permissions (required by XDG_RUNTIME_DIR spec)
	return os.MkdirTemp(dir, prefix)
}

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