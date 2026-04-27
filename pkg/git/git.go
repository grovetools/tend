package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Git represents a git repository
type Git struct {
	dir string
}

// New creates a new Git instance for the given directory
func New(dir string) *Git {
	return &Git{dir: dir}
}

// Init initializes a new git repository
func Init(dir string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = dir

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git init failed: %w\nOutput: %s", err, output)
	}

	return nil
}

// exec runs a git command in the repository directory
func (g *Git) exec(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s failed: %w\nStderr: %s",
			strings.Join(args, " "), err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// Add stages files for commit
func (g *Git) Add(patterns ...string) error {
	if len(patterns) == 0 {
		patterns = []string{"."}
	}

	args := append([]string{"add"}, patterns...)
	_, err := g.exec(args...)
	return err
}

// Commit creates a commit with the given message
func (g *Git) Commit(message string) error {
	_, err := g.exec("commit", "-m", message)
	return err
}

// AddCommit stages all changes and commits with the given message
func (g *Git) AddCommit(message string) error {
	if err := g.Add(); err != nil {
		return fmt.Errorf("staging files: %w", err)
	}

	if err := g.Commit(message); err != nil {
		// Check if there's nothing to commit
		if strings.Contains(err.Error(), "nothing to commit") {
			return nil
		}
		return fmt.Errorf("committing: %w", err)
	}

	return nil
}

// Status returns the current git status
func (g *Git) Status() (string, error) {
	return g.exec("status", "--porcelain")
}

// CurrentBranch returns the current branch name
func (g *Git) CurrentBranch() (string, error) {
	return g.exec("rev-parse", "--abbrev-ref", "HEAD")
}

// CreateBranch creates a new branch
func (g *Git) CreateBranch(name string) error {
	_, err := g.exec("checkout", "-b", name)
	return err
}

// Checkout switches to the specified branch
func (g *Git) Checkout(ref string) error {
	_, err := g.exec("checkout", ref)
	return err
}

// Convenience functions for direct usage without creating Git instance

// SetupTestConfig configures a repo with test defaults
func SetupTestConfig(dir string) error {
	g := New(dir)
	return g.SetUser("Test User", "test@example.com")
}

// Add stages files for commit
func Add(dir string, patterns ...string) error {
	g := New(dir)
	return g.Add(patterns...)
}

// Commit creates a commit with the given message
func Commit(dir string, message string) error {
	g := New(dir)
	return g.Commit(message)
}

// CurrentBranch returns the current branch name
func CurrentBranch(dir string) (string, error) {
	g := New(dir)
	return g.CurrentBranch()
}

// CreateBranch creates a new branch
func CreateBranch(dir string, name string) error {
	g := New(dir)
	return g.CreateBranch(name)
}

// Checkout switches to the specified branch
func Checkout(dir string, ref string) error {
	g := New(dir)
	return g.Checkout(ref)
}

// CreateWorktree creates a new git worktree
func CreateWorktree(repoDir string, branch string, worktreeDir string) error {
	g := New(repoDir)
	return g.CreateWorktree(worktreeDir, branch)
}
