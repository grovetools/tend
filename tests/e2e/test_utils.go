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

	// Walk up to find the project root (where grove config exists)
	currentDir := wd
	for {
		// Check for any grove config file
		if hasGroveConfig(currentDir) {
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

	return "", fmt.Errorf("could not find project root (no grove config found)")
}

// FindTendBinary finds the actual tend binary path (not the test runner).
// This is used for testing tend CLI commands like `tend list`.
func FindTendBinary() (string, error) {
	// Get the path of the current executable (tend-e2e)
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	// The tend binary should be in the same directory as tend-e2e
	binDir := filepath.Dir(execPath)
	tendPath := filepath.Join(binDir, "tend")

	if _, err := os.Stat(tendPath); err != nil {
		// Try looking relative to the working directory
		wd, _ := os.Getwd()
		// Walk up to find the project root
		currentDir := wd
		for {
			if hasGroveConfig(currentDir) {
				tendPath = filepath.Join(currentDir, "bin", "tend")
				if _, err := os.Stat(tendPath); err == nil {
					return tendPath, nil
				}
				break
			}
			parent := filepath.Dir(currentDir)
			if parent == currentDir {
				break
			}
			currentDir = parent
		}
		return "", fmt.Errorf("tend binary not found at %s or in project bin directory", tendPath)
	}

	return tendPath, nil
}

// hasGroveConfig checks if a directory contains a grove config file.
// Supports .yml, .yaml, and .toml formats.
func hasGroveConfig(dir string) bool {
	configNames := []string{
		"grove.yml",
		"grove.yaml",
		"grove.toml",
		".grove.yml",
		".grove.yaml",
		".grove.toml",
	}
	for _, name := range configNames {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}
