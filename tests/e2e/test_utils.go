// File: tests/e2e/test_utils.go
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindProjectBinary finds the project's main binary path.
// For the tend E2E suite, this will be `bin/tend-e2e`.
func FindProjectBinary() (string, error) {
	// First try to get the executable path from os.Args[0]
	// This works when the test is already running
	if len(os.Args) > 0 && os.Args[0] != "" {
		binPath, err := filepath.Abs(os.Args[0])
		if err == nil {
			return binPath, nil
		}
	}

	// Fallback: construct path relative to current working directory
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("could not get working directory: %w", err)
	}

	// Walk up to find the project root (where grove.yml exists)
	currentDir := wd
	for {
		groveYml := filepath.Join(currentDir, "grove.yml")
		if _, err := os.Stat(groveYml); err == nil {
			// Found project root
			binaryPath := filepath.Join(currentDir, "bin", "tend-e2e")
			if _, err := os.Stat(binaryPath); err == nil {
				return binaryPath, nil
			}
			return "", fmt.Errorf("e2e runner binary not found at %s. Run 'make build-e2e-runner'", binaryPath)
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			// Reached filesystem root
			break
		}
		currentDir = parent
	}

	return "", fmt.Errorf("could not find project root (no grove.yml found)")
}
