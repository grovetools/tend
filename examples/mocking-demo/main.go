package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/mattsolo1/grove-tend/pkg/app"
	"github.com/mattsolo1/grove-tend/pkg/assert"
	"github.com/mattsolo1/grove-tend/pkg/harness"
)

// GitWorkflowScenario demonstrates mocking git commands
var GitWorkflowScenario = &harness.Scenario{
	Name:        "git-workflow",
	Description: "Tests a git workflow using mocked git commands",
	Tags:        []string{"git", "mocking"},
	Steps: []harness.Step{
		// Setup mocks
		harness.SetupMocks(
			harness.Mock{CommandName: "git"},
		),
		
		// Create a test repository
		harness.NewStep("Create test directory", func(ctx *harness.Context) error {
			testDir := ctx.NewDir("git-test")
			ctx.Set("repo_dir", testDir)
			
			// Create the directory
			if err := os.MkdirAll(testDir, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			
			// Create a test file
			testFile := filepath.Join(testDir, "test.txt")
			return os.WriteFile(testFile, []byte("Hello from mock test!\n"), 0644)
		}),
		
		// Initialize git repository
		harness.NewStep("Initialize git repository", func(ctx *harness.Context) error {
			repoDir := ctx.GetString("repo_dir")
			cmd := ctx.Command("git", "init").Dir(repoDir)
			result := cmd.Run()
			
			ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
			if result.Error != nil {
				return fmt.Errorf("git init failed: %w", result.Error)
			}
			return nil
		}),
		
		// Check git status
		harness.NewStep("Check git status", func(ctx *harness.Context) error {
			repoDir := ctx.GetString("repo_dir")
			cmd := ctx.Command("git", "status").Dir(repoDir)
			result := cmd.Run()
			
			ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
			return assert.Contains(result.Stdout, "On branch main", "should be on main branch")
		}),
		
		// Stage and commit files
		harness.NewStep("Stage and commit files", func(ctx *harness.Context) error {
			repoDir := ctx.GetString("repo_dir")
			
			// Add files
			addCmd := ctx.Command("git", "add", ".").Dir(repoDir)
			addResult := addCmd.Run()
			if addResult.Error != nil {
				return fmt.Errorf("git add failed: %w", addResult.Error)
			}
			
			// Commit
			commitCmd := ctx.Command("git", "commit", "-m", "Initial commit from mock test").Dir(repoDir)
			commitResult := commitCmd.Run()
			
			ctx.ShowCommandOutput(commitCmd.String(), commitResult.Stdout, commitResult.Stderr)
			return assert.Contains(commitResult.Stdout, "Initial commit from mock test", "commit message should appear in output")
		}),
	},
}

// DockerScenario demonstrates mocking docker commands
var DockerScenario = &harness.Scenario{
	Name:        "docker-operations",
	Description: "Tests docker operations using mocked docker commands",
	Tags:        []string{"docker", "mocking"},
	Steps: []harness.Step{
		// Setup mocks
		harness.SetupMocks(
			harness.Mock{CommandName: "docker"},
		),
		
		// Check docker version
		harness.NewStep("Check docker version", func(ctx *harness.Context) error {
			cmd := ctx.Command("docker", "version")
			result := cmd.Run()
			
			ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
			return assert.Contains(result.Stdout, "Docker version", "should show docker version")
		}),
		
		// List docker images
		harness.NewStep("List docker images", func(ctx *harness.Context) error {
			cmd := ctx.Command("docker", "images")
			result := cmd.Run()
			
			ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
			return assert.Contains(result.Stdout, "REPOSITORY", "should show image list header")
		}),
		
		// Pull an image
		harness.NewStep("Pull docker image", func(ctx *harness.Context) error {
			cmd := ctx.Command("docker", "pull", "nginx:latest")
			result := cmd.Run()
			
			ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
			return assert.Contains(result.Stdout, "Downloaded newer image", "should simulate image download")
		}),
		
		// List running containers
		harness.NewStep("List running containers", func(ctx *harness.Context) error {
			cmd := ctx.Command("docker", "ps")
			result := cmd.Run()
			
			ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
			
			// Verify we see the mocked running container
			if err := assert.Contains(result.Stdout, "redis", "should show redis container"); err != nil {
				return err
			}
			return assert.Contains(result.Stdout, "Up 5 minutes", "should show container status")
		}),
	},
}

// LLMIntegrationScenario demonstrates using both inline scripts and binary mocks
var LLMIntegrationScenario = &harness.Scenario{
	Name:        "llm-integration",
	Description: "Tests LLM integration with different mock strategies",
	Tags:        []string{"llm", "mocking", "integration"},
	Steps: []harness.Step{
		// Setup binary mock for llm
		harness.SetupMocks(
			harness.Mock{CommandName: "llm"},
		),
		
		// Also demonstrate inline script mock
		harness.SetupMocks(
			harness.Mock{
				CommandName: "simple-tool",
				Script: `#!/bin/bash
echo "Simple tool output: $@"
echo "Environment: TEST_VAR=$TEST_VAR"
exit 0`,
			},
		),
		
		// Test the binary mock
		harness.NewStep("Query LLM with test prompt", func(ctx *harness.Context) error {
			cmd := ctx.Command("llm", "Tell me about testing")
			result := cmd.Run()
			
			ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
			return assert.Contains(result.Stdout, "testing", "LLM should respond about testing")
		}),
		
		// Test JSON output mode
		harness.NewStep("Query LLM with JSON output", func(ctx *harness.Context) error {
			cmd := ctx.Command("llm", "--json", "-p", "Generate some code")
			result := cmd.Run()
			
			ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
			
			// Verify JSON structure
			if err := assert.Contains(result.Stdout, `"prompt"`, "should have prompt field"); err != nil {
				return err
			}
			return assert.Contains(result.Stdout, `"response"`, "should have response field")
		}),
		
		// Test the inline script mock
		harness.NewStep("Test simple tool", func(ctx *harness.Context) error {
			cmd := ctx.Command("simple-tool", "arg1", "arg2").
				Env("TEST_VAR=mock-value")
			result := cmd.Run()
			
			ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
			
			if err := assert.Contains(result.Stdout, "arg1 arg2", "should echo arguments"); err != nil {
				return err
			}
			return assert.Contains(result.Stdout, "TEST_VAR=mock-value", "should show environment")
		}),
	},
}

// FlowMockScenario demonstrates swapping between mock and real flow
var FlowMockScenario = &harness.Scenario{
	Name:        "flow-mock-demo",
	Description: "Demonstrates mock/real binary swapping with grove flow",
	Tags:        []string{"flow", "real-deps"},
	Steps: []harness.Step{
		harness.SetupMocks(
			harness.Mock{CommandName: "flow"},
		),
		
		harness.NewStep("Check flow version", func(ctx *harness.Context) error {
			cmd := ctx.Command("flow", "version")
			result := cmd.Run()
			ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
			
			// Check if it's the mock
			if assert.Contains(result.Stdout, "mock", "should be using mock flow") != nil {
				fmt.Println("✓ Using mock flow")
			}
			return nil
		}),
		
		harness.NewStep("Show how to use real flow", func(ctx *harness.Context) error {
			fmt.Println("\nTo run with the real flow binary:")
			fmt.Println("  ./mocking-demo run flow-mock-demo --use-real-deps=flow")
			return nil
		}),
	},
}

// MixedDependenciesScenario demonstrates selective mock swapping
var MixedDependenciesScenario = &harness.Scenario{
	Name:        "mixed-dependencies",
	Description: "Tests using a mix of mocked and real dependencies",
	Tags:        []string{"integration", "real-deps"},
	Steps: []harness.Step{
		// Setup multiple mocks
		harness.SetupMocks(
			harness.Mock{CommandName: "git"},
			harness.Mock{CommandName: "docker"},
			harness.Mock{CommandName: "kubectl"},
		),
		
		// Create a test script that uses multiple tools
		harness.NewStep("Create integration script", func(ctx *harness.Context) error {
			scriptDir := ctx.NewDir("scripts")
			if err := os.MkdirAll(scriptDir, 0755); err != nil {
				return fmt.Errorf("failed to create scripts directory: %w", err)
			}
			scriptPath := filepath.Join(scriptDir, "deploy.sh")
			
			script := `#!/bin/bash
set -e

echo "=== Deployment Script ==="
echo

echo "1. Checking git status..."
git status

echo
echo "2. Checking Docker..."
docker version | head -1

echo
echo "3. Checking Kubernetes..."
kubectl version --client=true 2>/dev/null || echo "kubectl mock: ready"

echo
echo "=== All checks passed ==="
`
			
			if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
				return err
			}
			
			ctx.Set("script_path", scriptPath)
			return nil
		}),
		
		// Run the integration script with all mocks
		harness.NewStep("Run with all mocks", func(ctx *harness.Context) error {
			scriptPath := ctx.GetString("script_path")
			cmd := ctx.Command("bash", scriptPath)
			result := cmd.Run()
			
			ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
			
			// Verify all mocks were used
			if err := assert.Contains(result.Stdout, "On branch main", "git mock output"); err != nil {
				return err
			}
			if err := assert.Contains(result.Stdout, "Docker version", "docker mock output"); err != nil {
				return err
			}
			return assert.Contains(result.Stdout, "Mock kubectl version", "kubectl mock output")
		}),
		
		// Note: To test with real dependencies, users would run:
		// ./mocking-demo run mixed-dependencies --use-real-deps=git
		// or
		// ./mocking-demo run mixed-dependencies --use-real-deps=all
		harness.NewStep("Show how to use real deps", func(ctx *harness.Context) error {
			fmt.Println("\nTo run this scenario with real dependencies:")
			fmt.Println("  ./mocking-demo run mixed-dependencies --use-real-deps=git")
			fmt.Println("  ./mocking-demo run mixed-dependencies --use-real-deps=git,docker")
			fmt.Println("  ./mocking-demo run mixed-dependencies --use-real-deps=all")
			return nil
		}),
	},
}

func main() {
	// Collect all scenarios
	scenarios := []*harness.Scenario{
		GitWorkflowScenario,
		DockerScenario,
		LLMIntegrationScenario,
		FlowMockScenario,
		MixedDependenciesScenario,
	}

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt, shutting down...")
		cancel()
	}()

	// Execute the application
	if err := app.Execute(ctx, scenarios); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}