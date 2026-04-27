package command

import (
	"bytes"
	"io"
	"os/exec"
	"strings"
	"time"
)

// Command represents a command to execute
type Command struct {
	cmd     *exec.Cmd
	timeout time.Duration
	env     []string
	stdin   io.Reader

	// Output capture
	stdout bytes.Buffer
	stderr bytes.Buffer
}

// Result contains the output and status of a command execution
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
	Error    error
}

// New creates a new command
func New(name string, args ...string) *Command {
	return &Command{
		cmd:     exec.Command(name, args...),
		timeout: 30 * time.Second, // Default timeout
	}
}

// Dir sets the working directory
func (c *Command) Dir(dir string) *Command {
	c.cmd.Dir = dir
	return c
}

// Env adds environment variables
func (c *Command) Env(env ...string) *Command {
	c.env = append(c.env, env...)
	return c
}

// Timeout sets the command timeout
func (c *Command) Timeout(d time.Duration) *Command {
	c.timeout = d
	return c
}

// Stdin sets the standard input
func (c *Command) Stdin(r io.Reader) *Command {
	c.stdin = r
	return c
}

// String returns a string representation of the command
func (c *Command) String() string {
	parts := []string{c.cmd.Path}
	parts = append(parts, c.cmd.Args[1:]...)
	return strings.Join(parts, " ")
}

// Environment returns the environment variables set for this command
func (c *Command) Environment() []string {
	return c.env
}

// Start executes the command in the background.
func (c *Command) Start() (*Process, error) {
	return c.start()
}
