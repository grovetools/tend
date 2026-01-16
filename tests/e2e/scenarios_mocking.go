// File: tests/e2e/scenarios_mocking.go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/grovetools/tend/pkg/assert"
	"github.com/grovetools/tend/pkg/harness"
	"github.com/grovetools/tend/pkg/verify"
)

// GitWorkflowScenario demonstrates mocking git commands
var GitWorkflowScenario = harness.NewScenario(
	"git-workflow",
	"Tests a git workflow using mocked git commands",
	[]string{"git", "mocking"},
	[]harness.Step{
		harness.SetupMocks(harness.Mock{CommandName: "git"}),
		harness.NewStep("Create test directory", func(ctx *harness.Context) error {
			testDir := ctx.NewDir("git-test")
			ctx.Set("repo_dir", testDir)
			if err := os.MkdirAll(testDir, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			testFile := filepath.Join(testDir, "test.txt")
			return os.WriteFile(testFile, []byte("Hello from mock test!\n"), 0644)
		}),
		harness.NewStep("Initialize git repository", func(ctx *harness.Context) error {
			repoDir := ctx.GetString("repo_dir")
			cmd := ctx.Command("git", "init").Dir(repoDir)
			result := cmd.Run()
			ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
			return result.Error
		}),
		harness.NewStep("Stage and commit files", func(ctx *harness.Context) error {
			repoDir := ctx.GetString("repo_dir")
			addCmd := ctx.Command("git", "add", ".").Dir(repoDir)
			if addResult := addCmd.Run(); addResult.Error != nil {
				return fmt.Errorf("git add failed: %w", addResult.Error)
			}
			commitCmd := ctx.Command("git", "commit", "-m", "Initial commit").Dir(repoDir)
			commitResult := commitCmd.Run()
			ctx.ShowCommandOutput(commitCmd.String(), commitResult.Stdout, commitResult.Stderr)
			return ctx.Check("commit message appears in output",
				assert.Contains(commitResult.Stdout, "Initial commit"))
		}),
	},
)

// DockerScenario demonstrates mocking docker commands
var DockerScenario = harness.NewScenario(
	"docker-operations",
	"Tests docker operations using mocked docker commands",
	[]string{"docker", "mocking"},
	[]harness.Step{
		harness.SetupMocks(harness.Mock{CommandName: "docker"}),
		harness.NewStep("Verify docker commands", func(ctx *harness.Context) error {
			versionCmd := ctx.Command("docker", "version")
			versionResult := versionCmd.Run()
			ctx.ShowCommandOutput(versionCmd.String(), versionResult.Stdout, versionResult.Stderr)

			imagesCmd := ctx.Command("docker", "images")
			imagesResult := imagesCmd.Run()
			ctx.ShowCommandOutput(imagesCmd.String(), imagesResult.Stdout, imagesResult.Stderr)

			return ctx.Verify(func(v *verify.Collector) {
				v.Contains("shows docker version", versionResult.Stdout, "Docker version")
				v.Contains("shows image list header", imagesResult.Stdout, "REPOSITORY")
			})
		}),
	},
)

// LLMIntegrationScenario demonstrates using both inline scripts and binary mocks
var LLMIntegrationScenario = harness.NewScenario(
	"llm-integration",
	"Tests LLM integration with different mock strategies",
	[]string{"llm", "mocking", "integration"},
	[]harness.Step{
		harness.SetupMocks(harness.Mock{CommandName: "llm"}),
		harness.NewStep("Query LLM with test prompt", func(ctx *harness.Context) error {
			cmd := ctx.Command("llm", "Tell me about testing")
			result := cmd.Run()
			ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
			return ctx.Check("LLM responds about testing",
				assert.Contains(result.Stdout, "testing"))
		}),
	},
)

// FlowMockScenario demonstrates swapping between mock and real flow
var FlowMockScenario = harness.NewScenario(
	"flow-mock-demo",
	"Demonstrates mock/real binary swapping with grove flow",
	[]string{"flow", "real-deps"},
	[]harness.Step{
		harness.SetupMocks(harness.Mock{CommandName: "flow"}),
		harness.NewStep("Check flow version", func(ctx *harness.Context) error {
			cmd := ctx.Command("flow", "version")
			result := cmd.Run()
			ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
			if assert.Contains(result.Stdout, "mock", "should be using mock flow") != nil {
				fmt.Println("* Using mock flow")
			}
			return nil
		}),
	},
)

// MixedDependenciesScenario demonstrates selective mock swapping
var MixedDependenciesScenario = harness.NewScenario(
	"mixed-dependencies",
	"Tests using a mix of mocked and real dependencies",
	[]string{"integration", "real-deps"},
	[]harness.Step{
		harness.SetupMocks(
			harness.Mock{CommandName: "git"},
			harness.Mock{CommandName: "docker"},
			harness.Mock{CommandName: "kubectl"},
		),
		harness.NewStep("Create integration script", func(ctx *harness.Context) error {
			scriptDir := ctx.NewDir("scripts")
			if err := os.MkdirAll(scriptDir, 0755); err != nil {
				return fmt.Errorf("failed to create scripts directory: %w", err)
			}
			scriptPath := filepath.Join(scriptDir, "deploy.sh")
			script := `#!/bin/bash
set -e
echo "1. Checking git status..."
git status
echo "2. Checking Docker..."
docker version | head -1
echo "3. Checking Kubernetes..."
kubectl version --client=true 2>/dev/null || echo "kubectl mock: ready"
`
			if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
				return err
			}
			ctx.Set("script_path", scriptPath)
			return nil
		}),
		harness.NewStep("Run with all mocks", func(ctx *harness.Context) error {
			scriptPath := ctx.GetString("script_path")
			cmd := ctx.Command("bash", scriptPath)
			result := cmd.Run()
			ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)

			return ctx.Verify(func(v *verify.Collector) {
				v.Contains("git mock output present", result.Stdout, "On branch main")
				v.Contains("docker mock output present", result.Stdout, "Docker version")
				v.Contains("kubectl mock output present", result.Stdout, "Mock kubectl version")
			})
		}),
	},
)
