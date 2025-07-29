package command

import (
	"fmt"
	"os/exec"
	"strings"
)

// RunSimple runs a simple command and returns stdout
func RunSimple(name string, args ...string) (string, error) {
	cmd := New(name, args...)
	return cmd.Output()
}

// RunInDir runs a command in a specific directory
func RunInDir(dir, name string, args ...string) (string, error) {
	cmd := New(name, args...)
	cmd.Dir(dir)
	return cmd.Output()
}

// Exists checks if a command exists in PATH
func Exists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// MustRun runs a command and panics on error (for test setup)
func MustRun(name string, args ...string) string {
	output, err := RunSimple(name, args...)
	if err != nil {
		panic(fmt.Sprintf("command failed: %s %s: %v",
			name, strings.Join(args, " "), err))
	}
	return output
}

// RunWithEnv runs a command with additional environment variables
func RunWithEnv(env []string, name string, args ...string) (string, error) {
	cmd := New(name, args...)
	cmd.Env(env...)
	return cmd.Output()
}

// RunBash runs a bash command
func RunBash(script string) (string, error) {
	return RunSimple("bash", "-c", script)
}

// RunBashInDir runs a bash script in a specific directory
func RunBashInDir(dir, script string) (string, error) {
	return RunInDir(dir, "bash", "-c", script)
}