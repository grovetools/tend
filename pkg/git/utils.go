package git

import (
	"os/exec"
	"strings"
)

// IsGitInstalled checks if git is available in PATH
func IsGitInstalled() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// GetVersion returns the git version
func GetVersion() (string, error) {
	cmd := exec.Command("git", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// IsRepo checks if a directory is a git repository
func IsRepo(dir string) bool {
	g := New(dir)
	_, err := g.exec("rev-parse", "--git-dir")
	return err == nil
}

// GetRootDir returns the root directory of the git repository
func (g *Git) GetRootDir() (string, error) {
	return g.exec("rev-parse", "--show-toplevel")
}

// HasUncommittedChanges checks if there are uncommitted changes
func (g *Git) HasUncommittedChanges() (bool, error) {
	status, err := g.Status()
	if err != nil {
		return false, err
	}

	return status != "", nil
}
