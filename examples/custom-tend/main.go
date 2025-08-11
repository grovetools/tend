package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/mattsolo1/grove-tend/pkg/harness"
	"github.com/mattsolo1/grove-tend/pkg/app"
	"github.com/mattsolo1/grove-tend/pkg/command"
	"github.com/mattsolo1/grove-tend/pkg/fs"
	"github.com/mattsolo1/grove-tend/pkg/git"
)

// ExampleBasicScenario creates a basic example scenario for testing the framework
func ExampleBasicScenario() *harness.Scenario {
	return &harness.Scenario{
		Name:        "example-basic",
		Description: "A basic example scenario that demonstrates core framework features",
		Tags:        []string{"example", "smoke", "basic"},
		Steps: []harness.Step{
			{
				Name:        "Create test directory",
				Description: "Creates a temporary directory for testing",
				Func: func(ctx *harness.Context) error {
					testDir := ctx.NewDir("test")
					if err := fs.CreateDir(testDir); err != nil {
						return fmt.Errorf("failed to create test directory: %w", err)
					}
					ctx.Set("created_dir", testDir)
					return nil
				},
			},
			{
				Name:        "Write configuration file",
				Description: "Creates a basic Grove configuration file",
				Func: func(ctx *harness.Context) error {
					testDir := ctx.Dir("test")
					
					return fs.WriteBasicGroveConfig(testDir)
				},
			},
			{
				Name:        "Verify file exists",
				Description: "Confirms the configuration file was created successfully",
				Func: func(ctx *harness.Context) error {
					testDir := ctx.Dir("test")
					configPath := filepath.Join(testDir, "grove.yml")
					
					if !fs.Exists(configPath) {
						return fmt.Errorf("configuration file does not exist at %s", configPath)
					}
					
					content, err := fs.ReadString(configPath)
					if err != nil {
						return fmt.Errorf("failed to read configuration file: %w", err)
					}
					
					ctx.Set("config_content", content)
					return nil
				},
			},
			harness.DelayStep("Wait for filesystem", 100*time.Millisecond),
			{
				Name:        "Cleanup verification",
				Description: "Ensures cleanup will work properly",
				Func: func(ctx *harness.Context) error {
					testDir := ctx.Dir("test")
					if testDir == "" {
						return fmt.Errorf("test directory not found in context")
					}
					return nil
				},
			},
		},
	}
}

// ExampleGitScenario creates an example scenario that tests Git operations
func ExampleGitScenario() *harness.Scenario {
	return &harness.Scenario{
		Name:        "example-git",
		Description: "Example scenario demonstrating Git operations and worktree management",
		Tags:        []string{"example", "git", "integration"},
		Steps: []harness.Step{
			{
				Name:        "Setup test repository",
				Description: "Creates a new Git repository for testing",
				Func: func(ctx *harness.Context) error {
					repoDir := ctx.NewDir("repo")
					
					if err := git.Init(repoDir); err != nil {
						return fmt.Errorf("failed to initialize git repository: %w", err)
					}
					
					// Configure Git for testing
					if err := git.SetupTestConfig(repoDir); err != nil {
						return fmt.Errorf("failed to setup git config: %w", err)
					}
					
					ctx.Set("repo_dir", repoDir)
					return nil
				},
			},
			{
				Name:        "Create initial commit",
				Description: "Adds initial files and creates first commit",
				Func: func(ctx *harness.Context) error {
					repoDir := ctx.Dir("repo")
					
					// Create a simple file
					readmePath := filepath.Join(repoDir, "README.md")
					if err := fs.WriteString(readmePath, "# Test Repository\n\nThis is a test repository for Grove Tend testing.\n"); err != nil {
						return fmt.Errorf("failed to create README: %w", err)
					}
					
					// Add and commit
					if err := git.Add(repoDir, "README.md"); err != nil {
						return fmt.Errorf("failed to add file: %w", err)
					}
					
					if err := git.Commit(repoDir, "Initial commit"); err != nil {
						return fmt.Errorf("failed to create initial commit: %w", err)
					}
					
					return nil
				},
			},
			{
				Name:        "Create feature branch",
				Description: "Creates a new branch for feature development",
				Func: func(ctx *harness.Context) error {
					repoDir := ctx.Dir("repo")
					
					if err := git.CreateBranch(repoDir, "feature/test-branch"); err != nil {
						return fmt.Errorf("failed to create branch: %w", err)
					}
					
					if err := git.Checkout(repoDir, "feature/test-branch"); err != nil {
						return fmt.Errorf("failed to checkout branch: %w", err)
					}
					
					return nil
				},
			},
			{
				Name:        "Create worktree",
				Description: "Creates a Git worktree for parallel testing",
				Func: func(ctx *harness.Context) error {
					repoDir := ctx.Dir("repo")
					worktreeDir := ctx.NewDir("worktree")
					
					if err := git.CreateWorktree(repoDir, "main", worktreeDir); err != nil {
						return fmt.Errorf("failed to create worktree: %w", err)
					}
					
					ctx.Set("worktree_dir", worktreeDir)
					return nil
				},
			},
			{
				Name:        "Verify worktree isolation",
				Description: "Confirms worktree operates independently",
				Func: func(ctx *harness.Context) error {
					repoDir := ctx.Dir("repo")
					worktreeDir := ctx.Dir("worktree")
					
					// Check current branch in main repo
					mainBranch, err := git.CurrentBranch(repoDir)
					if err != nil {
						return fmt.Errorf("failed to get main repo branch: %w", err)
					}
					
					// Check current branch in worktree
					worktreeBranch, err := git.CurrentBranch(worktreeDir)
					if err != nil {
						return fmt.Errorf("failed to get worktree branch: %w", err)
					}
					
					if mainBranch == worktreeBranch {
						return fmt.Errorf("worktree should be on different branch: main=%s, worktree=%s", mainBranch, worktreeBranch)
					}
					
					return nil
				},
			},
		},
	}
}

// ExampleCommandScenario creates an example scenario that tests command execution
func ExampleCommandScenario() *harness.Scenario {
	return &harness.Scenario{
		Name:        "example-command",
		Description: "Example scenario demonstrating command execution and output capture",
		Tags:        []string{"example", "command", "shell"},
		Steps: []harness.Step{
			{
				Name:        "Run simple command",
				Description: "Executes a basic shell command",
				Func: func(ctx *harness.Context) error {
					cmd := command.New("echo", "Hello, Grove Tend!")
					result := cmd.Run()
					
					if result.Error != nil {
						return fmt.Errorf("command failed: %w", result.Error)
					}
					
					if result.ExitCode != 0 {
						return fmt.Errorf("expected exit code 0, got %d", result.ExitCode)
					}
					
					// Show actual command output
					ctx.ShowCommandOutput("echo Hello, Grove Tend!", result.Stdout, result.Stderr)
					
					ctx.Set("echo_output", result.Stdout)
					return nil
				},
			},
			{
				Name:        "Run command with timeout",
				Description: "Tests command execution with timeout handling",
				Func: func(ctx *harness.Context) error {
					cmd := command.New("sleep", "0.1").Timeout(1 * time.Second)
					result := cmd.Run()
					
					if result.Error != nil {
						return fmt.Errorf("sleep command should not timeout: %w", result.Error)
					}
					
					if result.Duration > 500*time.Millisecond {
						return fmt.Errorf("command took too long: %v", result.Duration)
					}
					
					return nil
				},
			},
			{
				Name:        "Run command in directory",
				Description: "Tests command execution in specific directory",
				Func: func(ctx *harness.Context) error {
					testDir := ctx.NewDir("cmdtest")
					if err := fs.CreateDir(testDir); err != nil {
						return fmt.Errorf("failed to create test directory: %w", err)
					}
					
					cmd := command.New("pwd").Dir(testDir)
					result := cmd.Run()
					
					if result.Error != nil {
						return fmt.Errorf("pwd command failed: %w", result.Error)
					}
					
					// Show actual command output
					ctx.ShowCommandOutput(fmt.Sprintf("pwd (in %s)", testDir), result.Stdout, result.Stderr)
					
					// Verify we're in the right directory
					outputDir := filepath.Clean(result.Stdout[:len(result.Stdout)-1]) // Remove newline
					expectedDir := filepath.Clean(testDir)
					
					if outputDir != expectedDir {
						return fmt.Errorf("expected directory %s, got %s", expectedDir, outputDir)
					}
					
					return nil
				},
			},
			harness.ConditionalStep("Optional Grove command test",
				func(ctx *harness.Context) bool {
					// Only run if Grove binary is available
					cmd := command.New("which", ctx.GroveBinary)
					result := cmd.Run()
					return result.Error == nil && result.ExitCode == 0
				},
				func(ctx *harness.Context) error {
					cmd := command.New(ctx.GroveBinary, "--version")
					result := cmd.Run()
					
					if result.Error != nil {
						return fmt.Errorf("grove --version failed: %w", result.Error)
					}
					
					// Show actual command output
					ctx.ShowCommandOutput(fmt.Sprintf("%s --version", ctx.GroveBinary), result.Stdout, result.Stderr)
					
					ctx.Set("grove_version", result.Stdout)
					return nil
				},
			),
		},
	}
}

// ExampleGroveVersionScenario creates a simple scenario that tests Grove binary availability
func ExampleGroveVersionScenario() *harness.Scenario {
	return &harness.Scenario{
		Name:        "example-grove-version",
		Description: "Simple test to verify Grove binary can be found and executed",
		Tags:        []string{"example", "smoke", "grove"},
		Steps: []harness.Step{
			{
				Name:        "Check Grove version",
				Description: "Runs grove --version to verify binary is available",
				Func: func(ctx *harness.Context) error {
					cmd := command.New(ctx.GroveBinary, "--version")
					result := cmd.Run()
					
					if result.Error != nil {
						return fmt.Errorf("grove --version failed: %w", result.Error)
					}
					
					if result.ExitCode != 0 {
						return fmt.Errorf("grove --version exited with code %d", result.ExitCode)
					}
					
					// Show the version output
					ctx.ShowCommandOutput(fmt.Sprintf("%s --version", ctx.GroveBinary), result.Stdout, result.Stderr)
					
					// Verify output contains expected content
					if !strings.Contains(result.Stdout, "Grove") {
						return fmt.Errorf("unexpected version output: %s", result.Stdout)
					}
					
					ctx.Set("grove_version_output", result.Stdout)
					return nil
				},
			},
			{
				Name:        "Check Grove help",
				Description: "Runs grove --help to verify basic functionality",
				Func: func(ctx *harness.Context) error {
					cmd := command.New(ctx.GroveBinary, "--help")
					result := cmd.Run()
					
					// Help command often returns exit code 0 or 1, both are acceptable
					if result.Error != nil && result.ExitCode != 1 {
						return fmt.Errorf("grove --help failed: %w", result.Error)
					}
					
					// Show help output (truncated for readability)
					output := result.Stdout
					if len(output) > 500 {
						output = output[:500] + "\n... (truncated)"
					}
					ctx.ShowCommandOutput(fmt.Sprintf("%s --help", ctx.GroveBinary), output, "")
					
					// Verify output contains expected content
					if !strings.Contains(result.Stdout, "Usage:") && !strings.Contains(result.Stdout, "Commands:") {
						return fmt.Errorf("help output doesn't contain expected content")
					}
					
					return nil
				},
			},
			{
				Name:        "Verify Grove binary path",
				Description: "Shows which Grove binary is being used",
				Func: func(ctx *harness.Context) error {
					// Display the binary path being used
					fmt.Printf("   Grove binary path: %s\n", ctx.GroveBinary)
					
					// If it's just "grove", try to find where it is in PATH
					if ctx.GroveBinary == "grove" {
						whichCmd := command.New("which", "grove")
						result := whichCmd.Run()
						if result.Error == nil && result.ExitCode == 0 {
							actualPath := strings.TrimSpace(result.Stdout)
							fmt.Printf("   Resolved from PATH: %s\n", actualPath)
							ctx.Set("grove_actual_path", actualPath)
						}
					}
					
					return nil
				},
			},
		},
	}
}

// TestKeywordFilteringScenario creates a scenario that tests the keyword filtering feature
func TestKeywordFilteringScenario() *harness.Scenario {
	return &harness.Scenario{
		Name:        "test-keyword-filtering",
		Description: "Tests the tend list --keyword functionality",
		Tags:        []string{"test", "filtering", "cli"},
		Steps: []harness.Step{
			{
				Name:        "Test keyword filtering for 'git'",
				Description: "Verify that --keyword=git returns only git-related scenarios",
				Func: func(ctx *harness.Context) error {
					// Use the current binary for testing
					tendBinary := os.Args[0]
					cmd := command.New(tendBinary, "list", "--keyword=git")
					result := cmd.Run()
					
					if result.Error != nil {
						return fmt.Errorf("tend list --keyword=git failed: %w", result.Error)
					}
					
					// Check that output contains example-git scenario
					if !strings.Contains(result.Stdout, "example-git") {
						return fmt.Errorf("expected 'example-git' in output, got: %s", result.Stdout)
					}
					
					// Check that output doesn't contain unrelated scenarios
					if strings.Contains(result.Stdout, "my-custom-scenario") {
						return fmt.Errorf("unexpected 'my-custom-scenario' in filtered output")
					}
					
					// Count scenarios in output (should be 1)
					if !strings.Contains(result.Stdout, "Available scenarios (1)") {
						return fmt.Errorf("expected exactly 1 scenario, output: %s", result.Stdout)
					}
					
					ctx.ShowCommandOutput("tend list --keyword=git", result.Stdout, result.Stderr)
					return nil
				},
			},
			{
				Name:        "Test keyword filtering for 'example'",
				Description: "Verify that --keyword=example returns multiple scenarios",
				Func: func(ctx *harness.Context) error {
					tendBinary := os.Args[0]
					cmd := command.New(tendBinary, "list", "--keyword=example")
					result := cmd.Run()
					
					if result.Error != nil {
						return fmt.Errorf("tend list --keyword=example failed: %w", result.Error)
					}
					
					// Should find multiple example scenarios
					if !strings.Contains(result.Stdout, "example-basic") {
						return fmt.Errorf("expected 'example-basic' in output")
					}
					if !strings.Contains(result.Stdout, "example-git") {
						return fmt.Errorf("expected 'example-git' in output")
					}
					if !strings.Contains(result.Stdout, "example-command") {
						return fmt.Errorf("expected 'example-command' in output")
					}
					
					// Should NOT find non-example scenarios
					if strings.Contains(result.Stdout, "my-custom-scenario") {
						return fmt.Errorf("unexpected 'my-custom-scenario' in filtered output")
					}
					
					return nil
				},
			},
			{
				Name:        "Test keyword filtering with short flag",
				Description: "Verify that -k works as shorthand for --keyword",
				Func: func(ctx *harness.Context) error {
					tendBinary := os.Args[0]
					cmd := command.New(tendBinary, "list", "-k", "custom")
					result := cmd.Run()
					
					if result.Error != nil {
						return fmt.Errorf("tend list -k custom failed: %w", result.Error)
					}
					
					// Should find custom scenario
					if !strings.Contains(result.Stdout, "my-custom-scenario") {
						return fmt.Errorf("expected 'my-custom-scenario' in output, got: %s", result.Stdout)
					}
					
					// Should be exactly 1 or 2 scenarios (my-custom-scenario and possibly test-keyword-filtering)
					scenarioCount := 0
					if strings.Contains(result.Stdout, "Available scenarios (1)") {
						scenarioCount = 1
					} else if strings.Contains(result.Stdout, "Available scenarios (2)") {
						scenarioCount = 2
					}
					
					if scenarioCount == 0 {
						return fmt.Errorf("unexpected scenario count in output: %s", result.Stdout)
					}
					
					return nil
				},
			},
			{
				Name:        "Test combining keyword and tag filters",
				Description: "Verify that --keyword and --tags work together",
				Func: func(ctx *harness.Context) error {
					tendBinary := os.Args[0]
					cmd := command.New(tendBinary, "list", "--tags=smoke", "--keyword=grove")
					result := cmd.Run()
					
					if result.Error != nil {
						return fmt.Errorf("tend list with combined filters failed: %w", result.Error)
					}
					
					// Should only find example-grove-version (has both 'smoke' tag and 'grove' in name)
					if !strings.Contains(result.Stdout, "example-grove-version") {
						return fmt.Errorf("expected 'example-grove-version' in output")
					}
					
					// Should not find example-basic (has smoke tag but no 'grove' keyword)
					if strings.Contains(result.Stdout, "example-basic") {
						return fmt.Errorf("unexpected 'example-basic' in filtered output")
					}
					
					// Should be exactly 1 scenario
					if !strings.Contains(result.Stdout, "Available scenarios (1)") {
						return fmt.Errorf("expected exactly 1 scenario with combined filters")
					}
					
					return nil
				},
			},
			{
				Name:        "Test case-insensitive keyword search",
				Description: "Verify that keyword search is case-insensitive",
				Func: func(ctx *harness.Context) error {
					tendBinary := os.Args[0]
					// Test uppercase keyword
					cmd := command.New(tendBinary, "list", "--keyword=COMMAND")
					result := cmd.Run()
					
					if result.Error != nil {
						return fmt.Errorf("tend list --keyword=COMMAND failed: %w", result.Error)
					}
					
					// Should still find example-command scenario
					if !strings.Contains(result.Stdout, "example-command") {
						return fmt.Errorf("case-insensitive search failed: expected 'example-command' with uppercase keyword")
					}
					
					return nil
				},
			},
		},
	}
}

// CustomScenario is an example of a scenario specific to this consumer
var CustomScenario = &harness.Scenario{
	Name:        "my-custom-scenario",
	Description: "A scenario defined in a different repository",
	Tags:        []string{"custom"},
	Steps: []harness.Step{
		harness.NewStep("Run custom logic", func(ctx *harness.Context) error {
			fmt.Println("This is a custom step from an external binary!")
			ctx.Set("custom_key", "custom_value")
			return nil
		}),
	},
}

func main() {
	// This example includes both the framework examples and custom scenarios
	// In a real repository, you would only include your custom scenarios
	myScenarios := []*harness.Scenario{
		// Example scenarios from the framework (for demonstration)
		ExampleBasicScenario(),
		ExampleGitScenario(),
		ExampleCommandScenario(),
		ExampleGroveVersionScenario(),
		
		// Custom scenarios specific to this repository
		CustomScenario,
		TestKeywordFilteringScenario(),
	}

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down...")
		cancel()
	}()

	// Execute the custom tend application
	if err := app.Execute(ctx, myScenarios); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}