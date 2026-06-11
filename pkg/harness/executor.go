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
	runtimeDir    string
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
		runtimeDir:    ctx.RuntimeDir(),
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

	// Inject sandboxed home environment.
	//
	// Note: we deliberately do NOT set GROVE_HOME. core's path resolution
	// (paths.getDataHome etc.) prefers GROVE_HOME over XDG_*, so a host
	// GROVE_HOME would override every XDG var below and make commands write
	// into the real grove data dir — a sandbox escape. By leaving it unset
	// (and stripping any inherited value in the preserve loop below) the XDG
	// vars take effect.
	env = append(env,
		fmt.Sprintf("HOME=%s", e.homeDir),
		fmt.Sprintf("XDG_CONFIG_HOME=%s", e.configDir),
		fmt.Sprintf("XDG_DATA_HOME=%s", e.dataDir),
		fmt.Sprintf("XDG_STATE_HOME=%s", e.stateDir),
		fmt.Sprintf("XDG_CACHE_HOME=%s", e.cacheDir),
		fmt.Sprintf("XDG_RUNTIME_DIR=%s", e.runtimeDir),
	)

	// Point GROVE_BIN at the test bin dir so core's paths.BinDir() (which
	// prefers GROVE_BIN) resolves the sandboxed mock binaries instead of any
	// host GROVE_BIN. When no mock bin dir exists, we leave GROVE_BIN unset so
	// BinDir() falls back to the sandboxed XDG_DATA_HOME/bin — either way
	// resolution stays inside the sandbox.
	if e.testBinDir != "" {
		env = append(env, fmt.Sprintf("GROVE_BIN=%s", e.testBinDir))
	}

	// Inject mock override environment variables
	for commandName, mockPath := range e.mockOverrides {
		envVarName := getOverrideEnvVarName(commandName)
		env = append(env, fmt.Sprintf("%s=%s", envVarName, mockPath))
	}

	// Preserve other environment variables. We strip PATH/HOME/XDG_* (set
	// above) and GROVE_HOME/GROVE_BIN (handled above) so no host value can
	// leak in and break sandbox isolation.
	for _, envVar := range os.Environ() {
		if !strings.HasPrefix(envVar, "PATH=") &&
			!strings.HasPrefix(envVar, "HOME=") &&
			!strings.HasPrefix(envVar, "XDG_") &&
			!strings.HasPrefix(envVar, "GROVE_HOME=") &&
			!strings.HasPrefix(envVar, "GROVE_BIN=") {
			env = append(env, envVar)
		}
	}

	cmd.Env = env
}
