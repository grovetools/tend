package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CreateWorktree creates a new git worktree
func (g *Git) CreateWorktree(path, branch string) error {
	// Ensure the parent directory exists
	parentDir := filepath.Dir(path)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}

	// Create the worktree
	_, err := g.exec("worktree", "add", path, "-b", branch)
	if err != nil {
		return fmt.Errorf("creating worktree: %w", err)
	}

	return nil
}

// RemoveWorktree removes a git worktree
func (g *Git) RemoveWorktree(name string) error {
	_, err := g.exec("worktree", "remove", name, "--force")
	return err
}

// ListWorktrees returns a list of all worktrees
func (g *Git) ListWorktrees() ([]string, error) {
	output, err := g.exec("worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	var worktrees []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			path := strings.TrimPrefix(line, "worktree ")
			worktrees = append(worktrees, path)
		}
	}

	return worktrees, nil
}

// PruneWorktrees removes references to deleted worktrees
func (g *Git) PruneWorktrees() error {
	_, err := g.exec("worktree", "prune")
	return err
}
