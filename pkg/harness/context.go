package harness

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	corecommand "github.com/grovetools/core/command"
	"github.com/grovetools/core/pkg/tmux"
	"github.com/grovetools/tend/pkg/command"
	"github.com/grovetools/tend/pkg/project"
	"github.com/grovetools/tend/pkg/teatest"
	"github.com/grovetools/tend/pkg/tui"
	"github.com/grovetools/tend/pkg/verify"
)

// contextMutex protects concurrent access to context maps
var contextMutex sync.RWMutex

// NewDir creates and tracks a named directory within the test.
// The directory is created on disk so callers can immediately chdir into it
// (or write files inside it) without an extra os.MkdirAll step.
func (c *Context) NewDir(name string) string {
	contextMutex.Lock()
	defer contextMutex.Unlock()

	dir := filepath.Join(c.RootDir, name)
	if c.dirs == nil {
		c.dirs = make(map[string]string)
	}
	c.dirs[name] = dir
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// Dir retrieves a previously created named directory
func (c *Context) Dir(name string) string {
	contextMutex.RLock()
	defer contextMutex.RUnlock()

	if c.dirs == nil {
		return ""
	}
	return c.dirs[name]
}

// Set stores a value for inter-step communication
func (c *Context) Set(key string, value interface{}) {
	contextMutex.Lock()
	defer contextMutex.Unlock()

	if c.values == nil {
		c.values = make(map[string]interface{})
	}
	c.values[key] = value
}

// Get retrieves a stored value
func (c *Context) Get(key string) interface{} {
	contextMutex.RLock()
	defer contextMutex.RUnlock()

	if c.values == nil {
		return nil
	}
	return c.values[key]
}

// GetString retrieves a stored string value
func (c *Context) GetString(key string) string {
	if v, ok := c.Get(key).(string); ok {
		return v
	}
	return ""
}

// GetStringSlice retrieves a stored string slice value
func (c *Context) GetStringSlice(key string) []string {
	if v, ok := c.Get(key).([]string); ok {
		return v
	}
	return nil
}

// GetInt retrieves a stored int value
func (c *Context) GetInt(key string) int {
	if v, ok := c.Get(key).(int); ok {
		return v
	}
	return 0
}

// ShowCommandOutput displays command output if UI is available and in verbose mode
func (c *Context) ShowCommandOutput(command, stdout, stderr string) {
	if c.ui != nil {
		c.ui.CommandOutput(command, stdout, stderr)
	}
}

// GetBool retrieves a stored bool value
func (c *Context) GetBool(key string) bool {
	if v, ok := c.Get(key).(bool); ok {
		return v
	}
	return false
}

// HasKey checks if a key exists in the context
func (c *Context) HasKey(key string) bool {
	contextMutex.RLock()
	defer contextMutex.RUnlock()

	if c.values == nil {
		return false
	}
	_, exists := c.values[key]
	return exists
}

// Keys returns all stored keys
func (c *Context) Keys() []string {
	contextMutex.RLock()
	defer contextMutex.RUnlock()

	if c.values == nil {
		return nil
	}

	keys := make([]string, 0, len(c.values))
	for k := range c.values {
		keys = append(keys, k)
	}
	return keys
}

// HomeDir returns the path to the sandboxed home directory for the test.
func (c *Context) HomeDir() string {
	return c.homeDir
}

// ConfigDir returns the path to the sandboxed XDG_CONFIG_HOME directory.
func (c *Context) ConfigDir() string {
	return c.configDir
}

// DataDir returns the path to the sandboxed XDG_DATA_HOME directory.
func (c *Context) DataDir() string {
	return c.dataDir
}

// StateDir returns the path to the sandboxed XDG_STATE_HOME directory.
func (c *Context) StateDir() string {
	return c.stateDir
}

// CacheDir returns the path to the sandboxed XDG_CACHE_HOME directory.
func (c *Context) CacheDir() string {
	return c.cacheDir
}

// RuntimeDir returns the path to the short XDG_RUNTIME_DIR.
// This uses /tmp directly to keep Unix socket paths under the ~104 char limit on macOS.
func (c *Context) RuntimeDir() string {
	return c.runtimeDir
}

// CommandExecutor creates a new TestExecutor configured with the current context.
// This is the preferred way to get a command executor within a test step, as it
// encapsulates all necessary test environment setup.
func (c *Context) CommandExecutor() corecommand.Executor {
	return NewTestExecutor(c)
}

// Command creates a new command with the test's mock-aware PATH.
func (c *Context) Command(program string, args ...string) *command.Command {
	binDir := c.GetString("test_bin_dir")
	finalProgramPath := program
	var constructedPath string

	// For non-absolute paths, we need to ensure mocks are found first
	if binDir != "" && !filepath.IsAbs(program) {
		// Check if the program exists in our mock bin directory
		mockPath := filepath.Join(binDir, program)
		if _, err := os.Stat(mockPath); err == nil {
			// Use the mock directly
			finalProgramPath = mockPath
		}
	}

	cmd := command.New(finalProgramPath, args...)

	// Construct and set the PATH environment variable
	if binDir != "" {
		currentPath := os.Getenv("PATH")
		constructedPath = fmt.Sprintf("PATH=%s:%s", binDir, currentPath)
		cmd.Env(constructedPath)

		// If in very verbose mode, log the path for debugging
		if c.ui != nil {
			c.ui.CommandOutput(fmt.Sprintf("PATH for '%s'", program), constructedPath, "")
		}
	}

	// Inject mock override environment variables
	for commandName, mockPath := range c.mockOverrides {
		envVarName := getOverrideEnvVarName(commandName)
		cmd.Env(fmt.Sprintf("%s=%s", envVarName, mockPath))
	}

	// Inject sandboxed home environment
	cmd.Env(
		fmt.Sprintf("HOME=%s", c.homeDir),
		fmt.Sprintf("XDG_CONFIG_HOME=%s", c.configDir),
		fmt.Sprintf("XDG_DATA_HOME=%s", c.dataDir),
		fmt.Sprintf("XDG_STATE_HOME=%s", c.stateDir),
		fmt.Sprintf("XDG_CACHE_HOME=%s", c.cacheDir),
		fmt.Sprintf("XDG_RUNTIME_DIR=%s", c.runtimeDir),
	)

	// Preserve or detect DOCKER_HOST to ensure Docker client can connect
	// even when HOME is sandboxed (which would make socket paths too long)
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" {
		// DOCKER_HOST not set, try to detect the real Docker socket
		// Use the real (unsandboxed) home directory for detection
		if realHome, err := os.UserHomeDir(); err == nil {
			// Try common Docker socket locations (Colima first, as it's most common on macOS)
			possibleSockets := []string{
				filepath.Join(realHome, ".colima/default/docker.sock"),
				filepath.Join(realHome, ".config/colima/default/docker.sock"),
				filepath.Join(realHome, ".docker/run/docker.sock"),
				"/var/run/docker.sock",
			}
			for _, socketPath := range possibleSockets {
				if _, err := os.Stat(socketPath); err == nil {
					dockerHost = fmt.Sprintf("unix://%s", socketPath)
					break
				}
			}
		}
	}
	if dockerHost != "" {
		cmd.Env(fmt.Sprintf("DOCKER_HOST=%s", dockerHost))
	}

	return cmd
}

// Bin creates a new command using the project binary under test.
// This is a convenience wrapper around Command() that automatically uses
// the binary path from grove.yml (stored in c.GroveBinary).
//
// Example:
//
//	cmd := ctx.Bin("plan", "init", "my-plan")  // instead of ctx.Command(flowBinary, "plan", "init", "my-plan")
func (c *Context) Bin(args ...string) *command.Command {
	// If GroveBinary is not a path (just "grove" or similar), try to resolve it
	if c.GroveBinary == "" || !filepath.IsAbs(c.GroveBinary) {
		// Try to find the binary via project.GetBinaryPath
		if c.ProjectRoot != "" {
			if binaryPath, err := project.GetBinaryPath(c.ProjectRoot); err == nil {
				c.GroveBinary = binaryPath
			}
		}
		// If still not resolved and is just "grove", this will use whatever is in PATH
	}
	return c.Command(c.GroveBinary, args...)
}

// getOverrideEnvVarName generates the environment variable name for a command override
func getOverrideEnvVarName(commandName string) string {
	// Special case for grove-hooks
	if commandName == "grove-hooks" {
		return "GROVE_HOOKS_BINARY"
	}
	// Generic pattern for others
	return fmt.Sprintf("GROVE_%s_BINARY", strings.ToUpper(strings.ReplaceAll(commandName, "-", "_")))
}

// StartTUI launches a TUI application in a new, isolated tmux session.
// It returns a Session handle for interaction and ensures the session is
// cleaned up automatically at the end of the scenario.
func (c *Context) StartTUI(binaryPath string, args []string, opts ...tui.StartOption) (*tui.Session, error) {
	// Process start options
	config := &tui.StartConfig{}
	for _, opt := range opts {
		opt(config)
	}

	// Convert relative paths to absolute paths
	if !filepath.IsAbs(binaryPath) {
		if absPath, err := filepath.Abs(binaryPath); err == nil {
			binaryPath = absPath
		}
	}

	// Automatically prepend mock binary directory to PATH if it exists.
	if binDir := c.GetString("test_bin_dir"); binDir != "" {
		pathVarFound := false
		// Search for and modify an existing PATH variable from user options.
		for i, envVar := range config.Env {
			if strings.HasPrefix(envVar, "PATH=") {
				// Prepend our mock dir to the user's custom PATH.
				config.Env[i] = "PATH=" + binDir + ":" + strings.TrimPrefix(envVar, "PATH=")
				pathVarFound = true
				break
			}
		}
		// If no PATH was provided by the user, create one based on the current process's PATH.
		if !pathVarFound {
			newPath := fmt.Sprintf("PATH=%s:%s", binDir, os.Getenv("PATH"))
			config.Env = append(config.Env, newPath)
		}
	}

	// Inject sandboxed home environment variables (HOME, XDG_CONFIG_HOME, etc.)
	// This ensures TUI processes use the test's isolated directories, similar to ctx.Command()
	config.Env = append(config.Env,
		fmt.Sprintf("HOME=%s", c.homeDir),
		fmt.Sprintf("XDG_CONFIG_HOME=%s", c.configDir),
		fmt.Sprintf("XDG_DATA_HOME=%s", c.dataDir),
		fmt.Sprintf("XDG_STATE_HOME=%s", c.stateDir),
		fmt.Sprintf("XDG_CACHE_HOME=%s", c.cacheDir),
		fmt.Sprintf("XDG_RUNTIME_DIR=%s", c.runtimeDir),
	)

	// If using isolated tmux socket, pass it to spawned processes
	// This ensures any tmux operations inside the TUI use the same isolated server
	if c.tmuxSocket != "" {
		config.Env = append(config.Env, fmt.Sprintf("GROVE_TMUX_SOCKET=%s", c.tmuxSocket))
	}

	// Set default color env vars so TUIs render colors in test sessions
	// These work with grove-core's tui.InitializeTUI() to enable colors
	colorEnvSet := false
	for _, env := range config.Env {
		if strings.HasPrefix(env, "CLICOLOR_FORCE=") {
			colorEnvSet = true
			break
		}
	}
	if !colorEnvSet {
		config.Env = append(config.Env,
			"CLICOLOR_FORCE=1",
			"COLORTERM=truecolor",
			"TERM=xterm-256color",
		)
	}

	// Create tmux client - use isolated socket if configured, otherwise default
	var tmuxClient *tmux.Client
	var err error

	if c.tmuxSocket != "" {
		tmuxClient, err = tmux.NewClientWithSocket(c.tmuxSocket)
	} else {
		tmuxClient, err = tmux.NewClient()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create tmux client for TUI session: %w", err)
	}

	// Generate a unique session name for test isolation
	sessionName := fmt.Sprintf("tend-tui-%s-%d", filepath.Base(binaryPath), time.Now().UnixNano())

	// Parse env vars from the config into a map
	// This separates environment variable setup from command execution,
	// avoiding long command strings that can get truncated with fish shell
	envMap := make(map[string]string)
	for _, envVar := range config.Env {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Build the command string *without* environment variables
	var cmdBuilder strings.Builder
	cmdBuilder.WriteString(binaryPath)
	if len(args) > 0 {
		cmdBuilder.WriteString(" ")
		cmdBuilder.WriteString(strings.Join(args, " "))
	}

	workingDir := c.RootDir
	if config.Cwd != "" {
		workingDir = config.Cwd
	}

	launchOpts := tmux.LaunchOptions{
		SessionName:      sessionName,
		WorkingDirectory: workingDir,
		WindowIndex:      -1, // Don't reorder window position
		Panes: []tmux.PaneOptions{
			{
				Command: cmdBuilder.String(),
				Env:     envMap,
			},
		},
	}

	// Launch the session in the background
	if err := tmuxClient.Launch(context.Background(), launchOpts); err != nil {
		return nil, fmt.Errorf("failed to launch TUI in tmux session '%s': %w", sessionName, err)
	}

	// Register for cleanup - track all sessions created
	c.Set("active_tui_session_name", sessionName) // Keep for backward compat
	sessions := c.GetStringSlice("tui_sessions")
	sessions = append(sessions, sessionName)
	c.Set("tui_sessions", sessions)

	// Create the session handle
	session := tui.NewSession(sessionName, tmuxClient, c.RootDir)

	// Start recording if configured
	if c.recordTUIDir != "" {
		// Ensure the recording directory exists
		if err := os.MkdirAll(c.recordTUIDir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create TUI recording directory: %w", err)
		}

		// Generate a unique recording path
		recordingPath := filepath.Join(c.recordTUIDir, fmt.Sprintf("%s-%d", sessionName, time.Now().Unix()))
		if err := session.StartRecording(recordingPath); err != nil {
			// Non-fatal: log but continue
			if c.ui != nil {
				c.ui.Error("Failed to start TUI recording", err)
			}
		}
	}

	return session, nil
}

// StartHeadless launches a BubbleTea model in a headless, non-tmux test runner.
// This is ideal for testing model logic and view output without the overhead of a full TUI session.
func (c *Context) StartHeadless(model tea.Model) *teatest.HeadlessSession {
	return teatest.NewHeadlessSession(model)
}

// AddAssertion logs a new assertion result for the current step.
func (c *Context) AddAssertion(description string, err error) {
	contextMutex.Lock()
	defer contextMutex.Unlock()

	result := &AssertionResult{
		Description: description,
		Success:     err == nil,
	}
	if err != nil {
		result.Error = err.Error()
	}
	c.assertions = append(c.assertions, result)
}

// getAssertions retrieves the assertion results for the current step.
func (c *Context) getAssertions() []*AssertionResult {
	contextMutex.RLock()
	defer contextMutex.RUnlock()
	return c.assertions
}

// clearAssertions resets the assertion log, called before each new step.
func (c *Context) clearAssertions() {
	contextMutex.Lock()
	defer contextMutex.Unlock()
	c.assertions = nil
}

// Check performs a hard assertion. It logs the result and returns an
// error immediately if the assertion fails, stopping the current step.
func (c *Context) Check(description string, err error) error {
	c.AddAssertion(description, err)
	if err != nil {
		// Return a wrapped error to preserve the original error's details
		// while providing the context of the description.
		return fmt.Errorf("%s: %w", description, err)
	}
	return nil
}

// Verify provides a scope for soft assertions. It collects all failures
// within the provided function and returns a single aggregated error at the end.
// All checks, pass or fail, are logged for detailed reporting.
func (c *Context) Verify(fn func(v *verify.Collector)) error {
	collector := verify.New(c)
	fn(collector)
	return collector.Check()
}
