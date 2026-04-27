package command

import (
	"fmt"
	"strings"
	"time"
)

// Grove provides helpers for running grove commands
type Grove struct {
	binary string
	dir    string
}

// NewGrove creates a new Grove command helper
func NewGrove(binary string) *Grove {
	return &Grove{binary: binary}
}

// InDir sets the working directory for grove commands
func (g *Grove) InDir(dir string) *Grove {
	return &Grove{
		binary: g.binary,
		dir:    dir,
	}
}

// Run executes a grove command
func (g *Grove) Run(args ...string) *Result {
	cmd := New(g.binary, args...)
	if g.dir != "" {
		cmd.Dir(g.dir)
	}
	return cmd.Run()
}

// Output runs a grove command and returns stdout
func (g *Grove) Output(args ...string) (string, error) {
	result := g.Run(args...)
	if result.Error != nil {
		return "", fmt.Errorf("grove %s failed: %w\nStderr: %s",
			strings.Join(args, " "), result.Error, result.Stderr)
	}
	return strings.TrimSpace(result.Stdout), nil
}

// AgentUp starts the grove agent
func (g *Grove) AgentUp(detached bool) error {
	args := []string{"agent", "up"}
	// Note: detached parameter ignored as grove agent up doesn't support -d flag

	// For now, we need to work around the fact that 'grove agent up' might
	// run in foreground. We'll use a short timeout and ignore timeout errors.
	cmd := New(g.binary, args...)
	if g.dir != "" {
		cmd.Dir(g.dir)
	}
	// Use a short timeout - if agent starts successfully, command should return quickly
	cmd.Timeout(5 * time.Second)

	result := cmd.Run()

	// If we got a timeout, that might be OK - the agent could be running in foreground
	if result.Error != nil && strings.Contains(result.Error.Error(), "timeout") {
		// Check if agent is actually running
		statusResult := g.Run("agent", "status")
		if statusResult.ExitCode == 0 {
			// Agent command succeeded, check output
			output := strings.ToLower(statusResult.Stdout)
			if strings.Contains(output, "running") || strings.Contains(output, "up") {
				// Agent is running, timeout was expected
				return nil
			}
		}
		// Include status info in error for debugging
		return fmt.Errorf("agent up timed out and status check failed (exit %d): %s",
			statusResult.ExitCode, statusResult.Stdout)
	}

	if result.Error != nil {
		return fmt.Errorf("agent up failed: %w\nStderr: %s",
			result.Error, result.Stderr)
	}
	return nil
}

// AgentDown stops the grove agent
func (g *Grove) AgentDown() error {
	result := g.Run("agent", "down")
	if result.Error != nil {
		return fmt.Errorf("agent down failed: %w\nStderr: %s",
			result.Error, result.Stderr)
	}
	return nil
}

// ServiceUp starts a service
func (g *Grove) ServiceUp(service string, detached bool) error {
	args := []string{"up", service}
	if detached {
		args = append(args, "-d")
	}

	result := g.Run(args...)
	if result.Error != nil {
		return fmt.Errorf("service up failed: %w\nStderr: %s",
			result.Error, result.Stderr)
	}
	return nil
}

// ServiceDown stops a service
func (g *Grove) ServiceDown(service string) error {
	result := g.Run("down", service)
	if result.Error != nil {
		return fmt.Errorf("service down failed: %w\nStderr: %s",
			result.Error, result.Stderr)
	}
	return nil
}

// Status gets grove status
func (g *Grove) Status() (string, error) {
	return g.Output("status")
}
