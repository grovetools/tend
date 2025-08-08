package agent

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattsolo1/grove-tend/internal/harness"
	"github.com/mattsolo1/grove-tend/pkg/command"
	"github.com/mattsolo1/grove-tend/pkg/fs"
	"github.com/mattsolo1/grove-tend/pkg/git"
)

// AgentIsolationScenario tests that agents in different worktrees are isolated
var AgentIsolationScenario = &harness.Scenario{
	Name:        "agent-isolation",
	Description: "Verify agents in different git worktrees run in isolated containers",
	Tags:        []string{"agent", "isolation", "worktree"},
	Steps: []harness.Step{
		preCleanup(),  // Ensure clean state before starting
		setupMainWorktree(),
		startMainAgent(),
		verifyMainAgent(),
		setupFeatureWorktree(),
		startFeatureAgent(),
		verifyBothAgentsRunning(),
		verifyAgentsAreIsolated(),
		cleanupAgents(),
	},
}

func preCleanup() harness.Step {
	return harness.NewStep(
		"Pre-test cleanup - remove any conflicting containers",
		func(ctx *harness.Context) error {
			// This ensures we start with a clean state, preventing "container name already in use" errors
			
			// List of patterns to clean up
			patterns := []string{
				"main-main-",
				"feature-feature-",
				"grove-main-",
				"grove-feature-",
			}
			
			for _, pattern := range patterns {
				// Find all containers matching the pattern
				dockerCmd := command.New("docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", pattern), "--format", "{{.Names}}")
				result := dockerCmd.Run()
				
				if result.Error == nil && strings.TrimSpace(result.Stdout) != "" {
					containerNames := strings.Fields(result.Stdout)
					
					// For grove-* patterns, only remove agent containers
					if strings.HasPrefix(pattern, "grove-") {
						var filteredNames []string
						for _, name := range containerNames {
							if strings.Contains(name, "agent") || strings.Contains(name, "test-service") {
								filteredNames = append(filteredNames, name)
							}
						}
						containerNames = filteredNames
					}
					
					if len(containerNames) > 0 {
						fmt.Printf("Pre-cleanup: removing %d containers matching pattern '%s'\n", len(containerNames), pattern)
						// Build the full command with all arguments
						args := []string{"rm", "-f"}
						args = append(args, containerNames...)
						rmCmd := command.New("docker", args...)
						rmResult := rmCmd.Run()
						if rmResult.Error != nil {
							// Log but don't fail - this is best effort cleanup
							fmt.Printf("Warning: failed to remove some containers: %v\n", rmResult.Error)
						} else {
							fmt.Printf("Successfully removed containers: %s\n", strings.Join(containerNames, ", "))
						}
					}
				}
			}
			
			// Give Docker a moment to clean up
			time.Sleep(1 * time.Second)
			
			return nil
		},
	)
}

func setupMainWorktree() harness.Step {
	return harness.NewStep(
		"Setup main worktree with grove.yml",
		func(ctx *harness.Context) error {
			mainDir := ctx.NewDir("main")

			// Create the directory first
			if err := fs.CreateDir(mainDir); err != nil {
				return fmt.Errorf("creating main directory: %w", err)
			}

			// Initialize git repository
			if err := git.Init(mainDir); err != nil {
				return fmt.Errorf("initializing git repo: %w", err)
			}

			repo, err := git.SetupTestRepo(mainDir)
			if err != nil {
				return fmt.Errorf("setting up git repo: %w", err)
			}

			// Create grove.yml
			if err := fs.WriteGroveConfig(mainDir, BasicGroveConfig()); err != nil {
				return fmt.Errorf("writing grove.yml: %w", err)
			}

			// Create test files
			for path, content := range TestFiles() {
				fullPath := filepath.Join(mainDir, path)
				// Ensure directory exists
				if err := fs.CreateDir(filepath.Dir(fullPath)); err != nil {
					return fmt.Errorf("creating directory for %s: %w", path, err)
				}
				if err := fs.WriteString(fullPath, content); err != nil {
					return fmt.Errorf("writing %s: %w", path, err)
				}
			}

			// Commit everything
			if err := repo.AddCommit("Initial commit"); err != nil {
				return fmt.Errorf("creating initial commit: %w", err)
			}

			// Store repo reference for later use
			ctx.Set("main_repo", repo)

			return nil
		},
	)
}

func startMainAgent() harness.Step {
	return harness.NewStep(
		"Start agent in main worktree",
		func(ctx *harness.Context) error {
			mainDir := ctx.Dir("main")
			grove := command.NewGrove(ctx.GroveBinary).InDir(mainDir)

			if err := grove.AgentUp(true); err != nil {
				return fmt.Errorf("starting main agent: %w", err)
			}

			// Wait for agent to be ready
			if err := WaitForAgent(ctx, mainDir, 30*time.Second); err != nil {
				return fmt.Errorf("waiting for main agent: %w", err)
			}

			return nil
		},
	)
}

func verifyMainAgent() harness.Step {
	return harness.NewStep(
		"Verify main agent is running",
		func(ctx *harness.Context) error {
			mainDir := ctx.Dir("main")

			// Get container name
			containerName, err := GetAgentContainerName(ctx, mainDir)
			if err != nil {
				return fmt.Errorf("getting main agent container: %w", err)
			}

			// Verify it contains "grove" and "agent" (the exact format may vary)
			if !strings.Contains(containerName, "grove") || !strings.Contains(containerName, "agent") {
				return fmt.Errorf("unexpected container name format: got %s, expected to contain 'grove' and 'agent'",
					containerName)
			}

			// Store for later comparison
			ctx.Set("main_container", containerName)

			// Verify container is actually running
			docker := command.NewDocker()
			containers, err := docker.ListContainers(fmt.Sprintf("name=%s", containerName))
			if err != nil {
				return fmt.Errorf("listing containers: %w", err)
			}

			if len(containers) == 0 {
				return fmt.Errorf("main agent container not found")
			}

			return nil
		},
	)
}

func setupFeatureWorktree() harness.Step {
	return harness.NewStep(
		"Create feature worktree",
		func(ctx *harness.Context) error {
			featureDir := ctx.NewDir("feature")

			repo := ctx.Get("main_repo").(*git.Git)

			// Create worktree
			if err := repo.CreateWorktree(featureDir, "feature-branch"); err != nil {
				return fmt.Errorf("creating worktree: %w", err)
			}

			// Make a change in the feature branch
			featureRepo := git.New(featureDir)
			testFile := filepath.Join(featureDir, "feature.txt")
			if err := fs.WriteString(testFile, "This is a feature branch"); err != nil {
				return fmt.Errorf("writing feature file: %w", err)
			}

			if err := featureRepo.AddCommit("Add feature file"); err != nil {
				return fmt.Errorf("committing feature changes: %w", err)
			}

			return nil
		},
	)
}

func startFeatureAgent() harness.Step {
	return harness.NewStep(
		"Start agent in feature worktree",
		func(ctx *harness.Context) error {
			featureDir := ctx.Dir("feature")
			grove := command.NewGrove(ctx.GroveBinary).InDir(featureDir)

			if err := grove.AgentUp(true); err != nil {
				return fmt.Errorf("starting feature agent: %w", err)
			}

			// Wait for agent to be ready
			if err := WaitForAgent(ctx, featureDir, 30*time.Second); err != nil {
				return fmt.Errorf("waiting for feature agent: %w", err)
			}

			return nil
		},
	)
}

func verifyBothAgentsRunning() harness.Step {
	return harness.NewStep(
		"Verify both agents are running",
		func(ctx *harness.Context) error {
			featureDir := ctx.Dir("feature")

			// Get feature container name
			containerName, err := GetAgentContainerName(ctx, featureDir)
			if err != nil {
				return fmt.Errorf("getting feature agent container: %w", err)
			}

			// Verify it contains "grove" and "agent" (the exact format may vary)
			if !strings.Contains(containerName, "grove") || !strings.Contains(containerName, "agent") {
				return fmt.Errorf("unexpected container name format: got %s, expected to contain 'grove' and 'agent'",
					containerName)
			}

			// Store for comparison
			ctx.Set("feature_container", containerName)

			// List all grove containers
			docker := command.NewDocker()
			containers, err := docker.ListContainers("name=grove-")
			if err != nil {
				return fmt.Errorf("listing grove containers: %w", err)
			}

			if len(containers) < 2 {
				return fmt.Errorf("expected at least 2 grove containers, found %d", len(containers))
			}

			return nil
		},
	)
}

func verifyAgentsAreIsolated() harness.Step {
	return harness.NewStep(
		"Verify agents are isolated",
		func(ctx *harness.Context) error {
			mainContainer := ctx.GetString("main_container")
			featureContainer := ctx.GetString("feature_container")

			if err := VerifyAgentIsolation(ctx, mainContainer, featureContainer); err != nil {
				return fmt.Errorf("verifying isolation: %w", err)
			}

			// Additional check: verify they have different ports
			docker := command.NewDocker()

			mainInfo, err := docker.ListContainers(fmt.Sprintf("name=%s", mainContainer))
			if err != nil || len(mainInfo) == 0 {
				return fmt.Errorf("getting main container info: %w", err)
			}

			featureInfo, err := docker.ListContainers(fmt.Sprintf("name=%s", featureContainer))
			if err != nil || len(featureInfo) == 0 {
				return fmt.Errorf("getting feature container info: %w", err)
			}

			// Could add more detailed port/network isolation checks here

			return nil
		},
	)
}

func cleanupAgents() harness.Step {
	return harness.NewStep(
		"Stop and forcefully remove all test agents",
		func(ctx *harness.Context) error {
			// First, forcefully remove any containers matching the test patterns
			// This is crucial to ensure a clean state even if previous runs failed
			
			// Find and remove all containers matching "main-main-" pattern
			dockerCmd := command.New("docker", "ps", "-a", "--filter", "name=main-main-", "--format", "{{.Names}}")
			result := dockerCmd.Run()
			if result.Error == nil && strings.TrimSpace(result.Stdout) != "" {
				containerNames := strings.Fields(result.Stdout)
				if len(containerNames) > 0 {
					// Build the full command with all arguments
					args := []string{"rm", "-f"}
					args = append(args, containerNames...)
					rmCmd := command.New("docker", args...)
					rmResult := rmCmd.Run()
					if rmResult.Error != nil {
						fmt.Printf("Warning: failed to remove main-main containers: %v\n", rmResult.Error)
					}
				}
			}

			// Find and remove all containers matching "feature-feature-" pattern
			dockerCmd = command.New("docker", "ps", "-a", "--filter", "name=feature-feature-", "--format", "{{.Names}}")
			result = dockerCmd.Run()
			if result.Error == nil && strings.TrimSpace(result.Stdout) != "" {
				containerNames := strings.Fields(result.Stdout)
				if len(containerNames) > 0 {
					// Build the full command with all arguments
					args := []string{"rm", "-f"}
					args = append(args, containerNames...)
					rmCmd := command.New("docker", args...)
					rmResult := rmCmd.Run()
					if rmResult.Error != nil {
						fmt.Printf("Warning: failed to remove feature-feature containers: %v\n", rmResult.Error)
					}
				}
			}

			// Also look for any grove-*-agent containers from the test directories
			// This catches containers with different naming patterns
			testPatterns := []string{"grove-main-", "grove-feature-"}
			for _, pattern := range testPatterns {
				dockerCmd = command.New("docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", pattern), "--format", "{{.Names}}")
				result = dockerCmd.Run()
				if result.Error == nil && strings.TrimSpace(result.Stdout) != "" {
					containerNames := strings.Fields(result.Stdout)
					// Only remove containers that also contain "agent" to avoid removing non-test containers
					var agentContainers []string
					for _, name := range containerNames {
						if strings.Contains(name, "agent") {
							agentContainers = append(agentContainers, name)
						}
					}
					if len(agentContainers) > 0 {
						// Build the full command with all arguments
						args := []string{"rm", "-f"}
						args = append(args, agentContainers...)
						rmCmd := command.New("docker", args...)
						rmCmd.Run() // Best effort, ignore errors
					}
				}
			}

			// Now try graceful shutdown for any remaining agents
			dirs := []string{"main", "feature"}
			for _, dir := range dirs {
				workDir := ctx.Dir(dir)
				if workDir == "" {
					continue // Skip if directory doesn't exist
				}
				grove := command.NewGrove(ctx.GroveBinary).InDir(workDir)

				// Best effort cleanup - don't fail the test if cleanup fails
				result := grove.Run("agent", "down")
				if result.Error != nil {
					// Log but don't fail
					fmt.Printf("Info: grove agent down in %s: %v (this is expected if containers were already removed)\n", dir, result.Error)
				}
			}

			return nil
		},
	)
}