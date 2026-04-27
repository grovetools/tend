package command

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// Run executes the command and waits for completion
func (c *Command) Run() *Result {
	start := time.Now()

	// Setup environment
	if len(c.env) > 0 {
		c.cmd.Env = append(os.Environ(), c.env...)
	}

	// Setup I/O
	c.cmd.Stdout = &c.stdout
	c.cmd.Stderr = &c.stderr
	if c.stdin != nil {
		c.cmd.Stdin = c.stdin
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	// Start the command
	if err := c.cmd.Start(); err != nil {
		return &Result{
			Error:    fmt.Errorf("failed to start command: %w", err),
			Duration: time.Since(start),
		}
	}

	// Wait for completion with context
	done := make(chan error, 1)
	go func() {
		done <- c.cmd.Wait()
	}()

	select {
	case err := <-done:
		result := &Result{
			Stdout:   c.stdout.String(),
			Stderr:   c.stderr.String(),
			Duration: time.Since(start),
		}

		// Extract exit code
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					result.ExitCode = status.ExitStatus()
				} else {
					result.ExitCode = 1
				}
			} else {
				result.ExitCode = -1
			}
			result.Error = err
		} else {
			result.ExitCode = 0
		}

		return result

	case <-ctx.Done():
		// Kill the process on timeout
		if c.cmd.Process != nil {
			_ = c.cmd.Process.Kill()
		}

		return &Result{
			Stdout:   c.stdout.String(),
			Stderr:   c.stderr.String(),
			Error:    fmt.Errorf("command timed out after %v", c.timeout),
			ExitCode: -1,
			Duration: time.Since(start),
		}
	}
}

// RunContext executes the command with a custom context
func (c *Command) RunContext(ctx context.Context) *Result {
	start := time.Now()

	// Create a new command with context
	cmdCtx := exec.CommandContext(ctx, c.cmd.Path, c.cmd.Args[1:]...)
	cmdCtx.Dir = c.cmd.Dir

	// Setup environment
	if len(c.env) > 0 {
		cmdCtx.Env = append(os.Environ(), c.env...)
	}

	// Setup I/O
	cmdCtx.Stdout = &c.stdout
	cmdCtx.Stderr = &c.stderr
	if c.stdin != nil {
		cmdCtx.Stdin = c.stdin
	}

	// Run the command
	err := cmdCtx.Run()

	result := &Result{
		Stdout:   c.stdout.String(),
		Stderr:   c.stderr.String(),
		Duration: time.Since(start),
	}

	// Handle context cancellation
	if ctx.Err() != nil {
		result.Error = fmt.Errorf("command cancelled: %w", ctx.Err())
		result.ExitCode = -1
		return result
	}

	// Extract exit code
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				result.ExitCode = status.ExitStatus()
			} else {
				result.ExitCode = 1
			}
		} else {
			result.ExitCode = -1
		}
		result.Error = err
	} else {
		result.ExitCode = 0
	}

	return result
}

// Output runs the command and returns stdout on success
func (c *Command) Output() (string, error) {
	result := c.Run()
	if result.Error != nil {
		return "", fmt.Errorf("%w\nStderr: %s", result.Error, result.Stderr)
	}
	return strings.TrimSpace(result.Stdout), nil
}

// CombinedOutput runs the command and returns combined stdout/stderr
func (c *Command) CombinedOutput() (string, error) {
	result := c.Run()
	combined := result.Stdout
	if result.Stderr != "" {
		if combined != "" {
			combined += "\n"
		}
		combined += result.Stderr
	}

	if result.Error != nil {
		return combined, result.Error
	}
	return combined, nil
}

// start is the internal implementation of the Start method.
func (c *Command) start() (*Process, error) {
	// Setup environment
	if len(c.env) > 0 {
		c.cmd.Env = append(os.Environ(), c.env...)
	}

	// Setup I/O
	c.cmd.Stdout = &c.stdout
	c.cmd.Stderr = &c.stderr
	if c.stdin != nil {
		c.cmd.Stdin = c.stdin
	}

	// Start the command
	if err := c.cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	process := &Process{
		Cmd:      c.cmd,
		PID:      c.cmd.Process.Pid,
		stdout:   &c.stdout,
		stderr:   &c.stderr,
		finished: make(chan error, 1),
	}

	// Wait for completion in a goroutine
	go func() {
		process.finished <- c.cmd.Wait()
	}()

	return process, nil
}
