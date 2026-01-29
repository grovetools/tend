package harness

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// TestExecutor creates commands with a test-aware environment, including a modified
// PATH for mock binaries and sandboxed home directories.
type TestExecutor struct {
	testBinDir    string
	homeDir       string
	configDir     string
	dataDir       string
	stateDir      string
	cacheDir      string
	mockOverrides map[string]string
}

// NewTestExecutor creates a new TestExecutor.
func NewTestExecutor(ctx *Context) *TestExecutor {
	return &TestExecutor{
		testBinDir:    ctx.GetString("test_bin_dir"),
		homeDir:       ctx.HomeDir(),
		configDir:     ctx.ConfigDir(),
		dataDir:       ctx.DataDir(),
		stateDir:      ctx.StateDir(),
		cacheDir:      ctx.CacheDir(),
		mockOverrides: ctx.mockOverrides,
	}
}

// Command creates a new exec.Cmd instance with the test environment applied.
func (e *TestExecutor) Command(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	e.applyTestEnvironment(cmd)
	return cmd
}

// CommandContext creates a new context-aware exec.Cmd with the test environment.
func (e *TestExecutor) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	e.applyTestEnvironment(cmd)
	return cmd
}

// applyTestEnvironment modifies an exec.Cmd to include the test environment.
func (e *TestExecutor) applyTestEnvironment(cmd *exec.Cmd) {
	var env []string
	currentPath := os.Getenv("PATH")

	// Prepend test bin directory to PATH
	if e.testBinDir != "" {
		newPath := fmt.Sprintf("%s:%s", e.testBinDir, currentPath)
		env = append(env, "PATH="+newPath)
	}

	// Inject sandboxed home environment
	env = append(env,
		fmt.Sprintf("HOME=%s", e.homeDir),
		fmt.Sprintf("XDG_CONFIG_HOME=%s", e.configDir),
		fmt.Sprintf("XDG_DATA_HOME=%s", e.dataDir),
		fmt.Sprintf("XDG_STATE_HOME=%s", e.stateDir),
		fmt.Sprintf("XDG_CACHE_HOME=%s", e.cacheDir),
	)

	// Inject mock override environment variables
	for commandName, mockPath := range e.mockOverrides {
		envVarName := getOverrideEnvVarName(commandName)
		env = append(env, fmt.Sprintf("%s=%s", envVarName, mockPath))
	}

	// Preserve other environment variables
	for _, envVar := range os.Environ() {
		if !strings.HasPrefix(envVar, "PATH=") &&
			!strings.HasPrefix(envVar, "HOME=") &&
			!strings.HasPrefix(envVar, "XDG_") {
			env = append(env, envVar)
		}
	}

	cmd.Env = env
}
