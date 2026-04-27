package command

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

// StreamHandler is called for each line of output
type StreamHandler func(line string, isStderr bool)

// StreamOptions configures output streaming
type StreamOptions struct {
	Handler       StreamHandler
	BufferSize    int
	CombineOutput bool
}

// RunStreaming executes the command with streaming output
func (c *Command) RunStreaming(opts StreamOptions) *Result {
	if opts.BufferSize == 0 {
		opts.BufferSize = 4096
	}

	// Create pipes for stdout and stderr
	stdoutPipe, err := c.cmd.StdoutPipe()
	if err != nil {
		return &Result{Error: fmt.Errorf("creating stdout pipe: %w", err)}
	}

	stderrPipe, err := c.cmd.StderrPipe()
	if err != nil {
		return &Result{Error: fmt.Errorf("creating stderr pipe: %w", err)}
	}

	// Setup environment
	if len(c.env) > 0 {
		c.cmd.Env = append(os.Environ(), c.env...)
	}

	// Setup stdin
	if c.stdin != nil {
		c.cmd.Stdin = c.stdin
	}

	// Start the command
	if err := c.cmd.Start(); err != nil {
		return &Result{Error: fmt.Errorf("starting command: %w", err)}
	}

	// Stream output
	var wg sync.WaitGroup
	wg.Add(2)

	go c.streamOutput(stdoutPipe, false, opts, &wg)
	go c.streamOutput(stderrPipe, true, opts, &wg)

	// Wait for streaming to complete
	wg.Wait()

	// Wait for command to complete
	err = c.cmd.Wait()

	result := &Result{
		Stdout: c.stdout.String(),
		Stderr: c.stderr.String(),
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

// streamOutput reads from a pipe and calls the handler for each line
func (c *Command) streamOutput(pipe io.ReadCloser, isStderr bool, opts StreamOptions, wg *sync.WaitGroup) {
	defer wg.Done()
	defer pipe.Close()

	scanner := bufio.NewScanner(pipe)
	scanner.Buffer(make([]byte, opts.BufferSize), opts.BufferSize)

	for scanner.Scan() {
		line := scanner.Text()

		// Store in buffer
		if isStderr {
			c.stderr.WriteString(line + "\n")
		} else {
			c.stdout.WriteString(line + "\n")
		}

		// Call handler
		if opts.Handler != nil {
			opts.Handler(line, isStderr)
		}
	}
}
