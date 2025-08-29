package harness

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/mattsolo1/grove-tend/pkg/command"
)

// contextMutex protects concurrent access to context maps
var contextMutex sync.RWMutex

// NewDir creates and tracks a named directory within the test
func (c *Context) NewDir(name string) string {
	contextMutex.Lock()
	defer contextMutex.Unlock()

	dir := filepath.Join(c.RootDir, name)
	if c.dirs == nil {
		c.dirs = make(map[string]string)
	}
	c.dirs[name] = dir
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

// Command creates a new command with the test's mock-aware PATH.
func (c *Context) Command(program string, args ...string) *command.Command {
	// For non-absolute paths, we need to ensure mocks are found first
	if binDir := c.GetString("test_bin_dir"); binDir != "" && !filepath.IsAbs(program) {
		// Check if the program exists in our mock bin directory
		mockPath := filepath.Join(binDir, program)
		if _, err := os.Stat(mockPath); err == nil {
			// Use the mock directly
			cmd := command.New(mockPath, args...)
			// Still set PATH for any subprocesses
			currentPath := os.Getenv("PATH")
			cmd.Env(fmt.Sprintf("PATH=%s:%s", binDir, currentPath))
			return cmd
		}
	}
	
	// Fall back to normal command creation
	cmd := command.New(program, args...)
	if binDir := c.GetString("test_bin_dir"); binDir != "" {
		currentPath := os.Getenv("PATH")
		cmd.Env(fmt.Sprintf("PATH=%s:%s", binDir, currentPath))
	}
	return cmd
}