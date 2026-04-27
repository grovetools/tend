package harness

import (
	"fmt"
	"os"
	"path/filepath"
)

// Mock defines a command to be mocked during a test scenario using a compiled binary.
type Mock struct {
	// The name of the command to be replaced (e.g., "git", "llm").
	CommandName string
	// Optional: The path to a binary to use as the mock.
	// If empty, tend will look for a binary named "mock-{CommandName}"
	// in a conventional location like `./bin/mock-{CommandName}` or
	// `./tests/mocks/bin/mock-{CommandName}`.
	BinaryPath string
}

// SetupMocksWithSubprocessSupport is like SetupMocks but also creates a wrapper
// shell script that forces PATH for subprocess commands. This is useful when
// the code under test spawns subprocesses via `sh -c` or similar, which may
// not inherit the modified PATH correctly.
func SetupMocksWithSubprocessSupport(mocks ...Mock) Step {
	return NewStep("Setup test environment with subprocess-safe mocks", func(ctx *Context) error {
		// First run the standard mock setup
		if err := SetupMocks(mocks...).Func(ctx); err != nil {
			return err
		}

		// Create a wrapper shell script that forces PATH
		mockBinDir := ctx.GetString("test_bin_dir")
		wrapperPath := filepath.Join(ctx.RootDir, "mock-shell")
		wrapperContent := fmt.Sprintf(`#!/bin/bash
# Wrapper shell that ensures mocks are on PATH for subprocess commands
export PATH=%s:$PATH
exec /bin/bash "$@"
`, mockBinDir)

		if err := os.WriteFile(wrapperPath, []byte(wrapperContent), 0o755); err != nil {
			return fmt.Errorf("failed to create shell wrapper: %w", err)
		}

		// Set SHELL to use our wrapper
		os.Setenv("SHELL", wrapperPath)
		ctx.Set("mock_shell_wrapper", wrapperPath)

		return nil
	})
}

// SetupMocks is a harness.Step that prepares a sandboxed `bin` directory
// with the specified mocks, making them available on the PATH for
// subsequent steps. For complex, stateful, or dynamic mocks, it is
// recommended to use compiled Go binaries rather than inline scripts.
func SetupMocks(mocks ...Mock) Step {
	return NewStep("Setup Mocks", func(ctx *Context) error {
		// Create a dedicated bin directory for the test run.
		mockBinDir := filepath.Join(ctx.RootDir, "test_bin")
		if err := os.MkdirAll(mockBinDir, 0o755); err != nil {
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
					return fmt.Errorf("could not find real binary for '%s': %w", mock.CommandName, err)
				}

				if err := os.Symlink(realBinaryPath, targetPath); err != nil {
					return fmt.Errorf("failed to symlink real binary for %s: %w", mock.CommandName, err)
				}
				// Log the swap for clarity in verbose mode.
				if ctx.ui != nil {
					ctx.ui.Info("Mock Swap", fmt.Sprintf("Using real binary for '%s' -> %s", mock.CommandName, realBinaryPath))
				}
				// Store the real binary path in mock overrides (for consistent PATH handling)
				ctx.mockOverrides[mock.CommandName] = targetPath
				continue // Move to the next mock
			}

			// Use a pre-compiled binary.
			sourcePath := mock.BinaryPath
			if sourcePath == "" {
				// Convention: look for mock-{name} in various standard locations
				// Try these paths in order:
				// 1. tests/e2e/tend/mocks/bin/mock-{name} (new E2E test location)
				// 2. tests/mocks/bin/mock-{name} (general test location)
				// 3. bin/mock-{name} (older convention)

				possiblePaths := []string{
					filepath.Join(ctx.ProjectRoot, "tests", "e2e", "tend", "mocks", "bin", "mock-"+mock.CommandName),
					filepath.Join(ctx.ProjectRoot, "tests", "mocks", "bin", "mock-"+mock.CommandName),
					filepath.Join(ctx.ProjectRoot, "bin", "mock-"+mock.CommandName),
				}

				for _, path := range possiblePaths {
					if _, err := os.Stat(path); err == nil {
						sourcePath = path
						break
					}
				}

				// If still not found, use the first path for the error message
				if sourcePath == "" {
					sourcePath = possiblePaths[0]
				}
			}

			// Create a symlink to the mock binary.
			if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
				return fmt.Errorf("mock binary for '%s' not found at %s. Did you run 'make build-mocks'?", mock.CommandName, sourcePath)
			}
			if err := os.Symlink(sourcePath, targetPath); err != nil {
				return fmt.Errorf("failed to symlink mock for %s: %w", mock.CommandName, err)
			}

			// Store the mock override path in the context
			ctx.mockOverrides[mock.CommandName] = targetPath
		}
		return nil
	})
}
