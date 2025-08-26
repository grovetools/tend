package command

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

// Process represents a running background command.
type Process struct {
	Cmd      *exec.Cmd
	PID      int
	stdout   *bytes.Buffer
	stderr   *bytes.Buffer
	finished chan error
}

// Wait blocks until the process completes or the timeout is reached.
func (p *Process) Wait(timeout time.Duration) *Result {
	start := time.Now()

	// Create a context with the specified timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case err := <-p.finished:
		// Process finished on its own
		result := &Result{
			Stdout:   p.stdout.String(),
			Stderr:   p.stderr.String(),
			Duration: time.Since(start),
		}
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					result.ExitCode = status.ExitStatus()
				}
			}
			result.Error = err
		}
		return result

	case <-ctx.Done():
		// Timeout reached, kill the process
		p.Kill()
		return &Result{
			Stdout:   p.stdout.String(),
			Stderr:   p.stderr.String(),
			Error:    fmt.Errorf("process timed out after %v", timeout),
			ExitCode: -1,
			Duration: time.Since(start),
		}
	}
}

// Stdout returns the standard output captured so far.
func (p *Process) Stdout() string {
	return p.stdout.String()
}

// Stderr returns the standard error captured so far.
func (p *Process) Stderr() string {
	return p.stderr.String()
}

// Kill terminates the process.
func (p *Process) Kill() error {
	if p.Cmd.Process != nil {
		return p.Cmd.Process.Kill()
	}
	return fmt.Errorf("process not started")
}