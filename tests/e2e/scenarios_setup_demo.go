// File: tests/e2e/scenarios_setup_demo.go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/grovetools/tend/pkg/fs"
	"github.com/grovetools/tend/pkg/harness"
)

// setupTestWorkspace is a reusable setup step for scenarios that need a mock filesystem.
var setupTestWorkspace = harness.NewStep("Setup mock filesystem with test files", func(ctx *harness.Context) error {
	// Create a test directory structure
	testDir := ctx.NewDir("test-workspace")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		return fmt.Errorf("failed to create test workspace: %w", err)
	}

	// Create some test files
	configFile := filepath.Join(testDir, "config.yml")
	if err := fs.WriteString(configFile, "name: test-project\nversion: 1.0.0\n"); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	dataFile := filepath.Join(testDir, "data.txt")
	if err := fs.WriteString(dataFile, "Sample test data\n"); err != nil {
		return fmt.Errorf("failed to create data file: %w", err)
	}

	ctx.Set("test_workspace", testDir)
	return nil
})

// SetupDemoScenario demonstrates the new setup/teardown functionality.
// This scenario uses the WithSetup builder method to add a reusable setup step.
func SetupDemoScenario() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"setup-demo",
		"Demonstrates the setup/teardown lifecycle with reusable setup steps",
		[]string{"demo", "setup"},
		[]harness.Step{
			harness.NewStep("Verify workspace exists", func(ctx *harness.Context) error {
				workspace := ctx.GetString("test_workspace")
				if workspace == "" {
					return fmt.Errorf("workspace not set by setup step")
				}

				// Verify the directory exists
				if _, err := os.Stat(workspace); os.IsNotExist(err) {
					return fmt.Errorf("workspace directory does not exist: %s", workspace)
				}

				return nil
			}),
			harness.NewStep("Verify test files exist", func(ctx *harness.Context) error {
				workspace := ctx.GetString("test_workspace")

				configFile := filepath.Join(workspace, "config.yml")
				if _, err := os.Stat(configFile); os.IsNotExist(err) {
					return fmt.Errorf("config.yml does not exist")
				}

				dataFile := filepath.Join(workspace, "data.txt")
				if _, err := os.Stat(dataFile); os.IsNotExist(err) {
					return fmt.Errorf("data.txt does not exist")
				}

				return nil
			}),
		},
		false, // localOnly
		false, // explicitOnly
	).WithSetup(setupTestWorkspace)
}

// SetupDemoWithTeardownScenario demonstrates both setup and teardown phases.
func SetupDemoWithTeardownScenario() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"setup-teardown-demo",
		"Demonstrates setup and teardown phases working together",
		[]string{"demo", "setup", "teardown"},
		[]harness.Step{
			harness.NewStep("Create temporary resource marker", func(ctx *harness.Context) error {
				workspace := ctx.GetString("test_workspace")
				markerFile := filepath.Join(workspace, "active.marker")
				if err := fs.WriteString(markerFile, "Resource is active\n"); err != nil {
					return fmt.Errorf("failed to create marker: %w", err)
				}
				ctx.Set("marker_file", markerFile)
				return nil
			}),
			harness.NewStep("Verify marker exists", func(ctx *harness.Context) error {
				markerFile := ctx.GetString("marker_file")
				if _, err := os.Stat(markerFile); os.IsNotExist(err) {
					return fmt.Errorf("marker file does not exist")
				}
				return nil
			}),
		},
		false, // localOnly
		false, // explicitOnly
	).WithSetup(setupTestWorkspace).WithTeardown(
		harness.NewStep("Cleanup temporary marker", func(ctx *harness.Context) error {
			markerFile := ctx.GetString("marker_file")
			if markerFile != "" {
				if err := os.Remove(markerFile); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("failed to remove marker: %w", err)
				}
			}
			return nil
		}),
	)
}
