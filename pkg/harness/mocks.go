package harness

import (
	"fmt"
	"os"
	"path/filepath"
)

// Mock defines a command to be mocked during a test scenario.
type Mock struct {
	// The name of the command to be replaced (e.g., "git", "llm").
	CommandName string
	// Optional: The path to a binary to use as the mock.
	// If empty, tend will look for a binary named "mock-{CommandName}"
	// in the project's ./bin directory.
	BinaryPath string
	// Optional: An inline script for simple mocks. This provides a
	// convenient way to create simple mocks without a separate Go program
	// and maintains compatibility with the old approach.
	Script string
}

// SetupMocks is a harness.Step that prepares a sandboxed `bin` directory
// with the specified mocks, making them available on the PATH for
// subsequent steps.
func SetupMocks(mocks ...Mock) Step {
	return NewStep("Setup Mocks", func(ctx *Context) error {
		// Create a dedicated bin directory for the test run.
		mockBinDir := filepath.Join(ctx.RootDir, "test_bin")
		if err := os.MkdirAll(mockBinDir, 0755); err != nil {
			return fmt.Errorf("failed to create mock bin directory: %w", err)
		}

		// Store its path in the context for command runners to use.
		ctx.Set("test_bin_dir", mockBinDir)

		for _, mock := range mocks {
			targetPath := filepath.Join(mockBinDir, mock.CommandName)

			// Check if we should swap this mock for the real binary.
			swap := ctx.UseRealDeps["all"] || ctx.UseRealDeps[mock.CommandName]

			if swap {
				realBinaryPath, err := FindRealBinary(mock.CommandName)
				if err != nil {
					// Fail fast: if the user asked for a real binary, it must be available.
					return fmt.Errorf("could not find real binary for '%s' via 'grove dev current': %w", mock.CommandName, err)
				}

				if err := os.Symlink(realBinaryPath, targetPath); err != nil {
					return fmt.Errorf("failed to symlink real binary for %s: %w", mock.CommandName, err)
				}
				// Log the swap for clarity in verbose mode.
				if ctx.ui != nil {
					ctx.ui.Info("Mock Swap", fmt.Sprintf("Using real binary for '%s' -> %s", mock.CommandName, realBinaryPath))
				}
				continue // Move to the next mock
			}

			if mock.Script != "" {
				// Write inline script to the mock path.
				if err := os.WriteFile(targetPath, []byte(mock.Script), 0755); err != nil {
					return fmt.Errorf("failed to write mock script for %s: %w", mock.CommandName, err)
				}
			} else {
				// Use a pre-compiled binary.
				sourcePath := mock.BinaryPath
				if sourcePath == "" {
					// Convention: look for mock-{name} in project's bin dir.
					projectBinDir := filepath.Join(ctx.ProjectRoot, "bin")
					sourcePath = filepath.Join(projectBinDir, "mock-"+mock.CommandName)
				}

				// Create a symlink to the mock binary.
				if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
					return fmt.Errorf("mock binary for '%s' not found at %s. Did you run 'make build-mocks'?", mock.CommandName, sourcePath)
				}
				if err := os.Symlink(sourcePath, targetPath); err != nil {
					return fmt.Errorf("failed to symlink mock for %s: %w", mock.CommandName, err)
				}
			}
		}
		return nil
	})
}