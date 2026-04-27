package git

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/grovetools/tend/pkg/fs"
)

// CreateTestRepo creates a git repository with initial content
func CreateTestRepo(dir string, files map[string]string) (*Git, error) {
	// Setup the repository
	g, err := SetupTestRepo(dir)
	if err != nil {
		return nil, err
	}

	// Create initial files
	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		if err := fs.WriteString(fullPath, content); err != nil {
			return nil, fmt.Errorf("writing %s: %w", path, err)
		}
	}

	// Create initial commit
	if err := g.AddCommit("Initial commit"); err != nil {
		return nil, fmt.Errorf("creating initial commit: %w", err)
	}

	return g, nil
}

// CreateBranchWithChanges creates a new branch and makes changes
func (g *Git) CreateBranchWithChanges(branch string, changes map[string]string) error {
	// Create and checkout the branch
	if err := g.CreateBranch(branch); err != nil {
		return fmt.Errorf("creating branch: %w", err)
	}

	// Apply changes
	for path, content := range changes {
		fullPath := filepath.Join(g.dir, path)
		if err := fs.WriteString(fullPath, content); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}

	// Commit changes
	if err := g.AddCommit(fmt.Sprintf("Changes on %s", branch)); err != nil {
		return fmt.Errorf("committing changes: %w", err)
	}

	return nil
}

// CloneRepo clones a repository (for testing remote operations)
func CloneRepo(source, dest string) (*Git, error) {
	cmd := exec.Command("git", "clone", source, dest)

	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git clone failed: %w\nOutput: %s", err, output)
	}

	return New(dest), nil
}
