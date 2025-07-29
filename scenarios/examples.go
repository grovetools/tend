package scenarios

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/grovepm/grove-tend/internal/harness"
	"github.com/grovepm/grove-tend/pkg/command"
	"github.com/grovepm/grove-tend/pkg/fs"
	"github.com/grovepm/grove-tend/pkg/git"
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