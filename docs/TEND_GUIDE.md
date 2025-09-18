# Grove Tend Testing Library - Comprehensive Guide

A Go library for creating powerful, scenario-based end-to-end testing frameworks. Grove Tend provides the building blocks to replace ad-hoc bash scripts with structured, maintainable, and debuggable Go code. It is designed as a pure library, allowing you to build a custom testing CLI tailored to your project's needs. Its library-first design keeps test definitions close to your code, while features like a rich set of helper packages, interactive debugging, and first-class mocking support streamline the E2E testing process.

## Core Concepts

### Scenario

A Scenario is the fundamental unit of a test in Grove Tend. It represents a complete end-to-end test case, composed of a series of logical steps. Each scenario has a name, description, and tags, which help in organizing, filtering, and understanding the purpose of the test.

```go
// A basic scenario definition
        var MyWebAppScenario = &harness.Scenario{
            Name:        "webapp-smoke-test",
            Description: "Performs a basic smoke test on the web application.",
            Tags:        []string{"smoke", "webapp"},
            Steps: []harness.Step{
                // ... steps go here ...
            },
        }
```

### Step

A Step is a single, atomic action within a Scenario. Each step has a name and a function that contains the test logic. Steps are executed sequentially, and if any step fails, the scenario stops. The framework provides step builders for common actions like delays or conditional execution.

```go
// A basic step definition within a scenario's Steps slice
        harness.NewStep("Create test directory", func(ctx *harness.Context) error {
            // The context manages a temporary directory for the scenario
            testDir := ctx.NewDir("webapp-test")
            ctx.Set("test_dir", testDir) // Store values for later steps
            return fs.WriteBasicGroveConfig(testDir)
        }),
```

### Context

The Context is a state container passed between steps in a scenario. It manages the temporary test directory (`RootDir`) and provides a key-value store for sharing data (like file paths or command output) between steps. It also provides a mock-aware `Command()` factory for executing commands.

```go
// Using the context to manage state between steps
        var MyScenario = &harness.Scenario{
            Name: "context-demo",
            Steps: []harness.Step{
                harness.NewStep("Step 1: Create a file", func(ctx *harness.Context) error {
                    // Create a temporary directory managed by the harness
                    tempDir := ctx.NewDir("my-files")
                    filePath := filepath.Join(tempDir, "data.txt")
                    
                    // Store the file path for the next step
                    ctx.Set("data_file_path", filePath)
                    
                    return fs.WriteString(filePath, "hello from step 1")
                }),
                harness.NewStep("Step 2: Read the file", func(ctx *harness.Context) error {
                    // Retrieve the file path from the context
                    filePath := ctx.GetString("data_file_path")
                    
                    content, err := fs.ReadString(filePath)
                    if err != nil {
                        return err
                    }
                    
                    return assert.Contains(content, "hello from step 1")
                }),
            },
        }
```

## Usage Patterns

### Basic File Operations

The `fs` helper package provides a convenient way to perform common file system operations within a temporary, isolated test directory managed by the harness context.

```go
harness.NewStep("Setup test project structure", func(ctx *harness.Context) error {
            // Create a file with content
            if err := fs.WriteString(filepath.Join(ctx.RootDir, "main.go"), "package main"); err != nil {
                return err
            }
            // Create a directory
            if err := fs.CreateDir(filepath.Join(ctx.RootDir, "src")); err != nil {
                return err
            }
            // Check if a file exists
            if !fs.Exists(filepath.Join(ctx.RootDir, "main.go")) {
                return fmt.Errorf("file was not created")
            }
            return nil
        }),
```

### Command Execution

The `command` package simplifies running external commands. Using `ctx.Command()` ensures that any configured mocks are used correctly. The result includes stdout, stderr, exit code, and execution duration.

```go
harness.NewStep("Run a command", func(ctx *harness.Context) error {
            // Use ctx.Command to respect mock PATH
            cmd := ctx.Command("echo", "Hello, Tend!").Dir(ctx.RootDir)
            result := cmd.Run()
            if result.Error != nil {
                return result.Error
            }
            
            // Display formatted command output in verbose/debug mode
            ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
            
            // Use the assert package for verifications
            return assert.Contains(result.Stdout, "Hello, Tend!")
        }),
```

### Git Operations

The `git` package provides wrappers for common Git operations, making it easy to set up repositories, create commits, manage branches, and handle worktrees for testing.

```go
harness.NewStep("Setup git repository", func(ctx *harness.Context) error {
            repoDir := ctx.NewDir("my-repo")
            
            // Init repository and set test user config
            git.Init(repoDir)
            git.SetupTestConfig(repoDir)
            
            // Create a file, add, and commit
            fs.WriteString(filepath.Join(repoDir, "README.md"), "Initial commit")
            git.Add(repoDir, "README.md")
            git.Commit(repoDir, "feat: initial commit")
            
            // Create a worktree
            worktreePath := filepath.Join(ctx.RootDir, ".grove-worktrees", "feature-branch")
            return git.CreateWorktree(repoDir, "feature-branch", worktreePath)
        }),
```

### Mocking Dependencies

Tend provides a first-class mocking system. You define mocks as Go binaries, and the `harness.SetupMocks` step builder creates a sandboxed PATH where your mock binaries are used instead of the real ones. Use `ctx.Command()` to ensure commands are resolved correctly.

```go
// In your scenario definition
        var MyScenario = &harness.Scenario{
            Name: "my-feature-test",
            Steps: []harness.Step{
                // By convention, tend looks for ./bin/mock-git and ./bin/mock-llm.
                harness.SetupMocks(
                    harness.Mock{CommandName: "git"},
                    harness.Mock{CommandName: "llm"},
                ),
                harness.NewStep("Run feature command", func(ctx *harness.Context) error {
                    // This will execute ./bin/mock-git instead of the real git
                    cmd := ctx.Command("git", "status")
                    result := cmd.Run()
                    // ... assert on mock git output ...
                    return result.Error
                }),
            },
        }
```

### Swapping Mocks for Real Dependencies

For integration testing, you can selectively swap mocks for their real counterparts using the `--use-real-deps` flag. Tend uses `grove dev current <tool>` to discover the path to the active binary in your ecosystem.

```bash
# Run with all mocks enabled (default)
        ./my-tests run my-feature-test

        # Swap the 'git' mock for the real binary found by grove
        ./my-tests run my-feature-test --use-real-deps=git

        # Swap 'git' and 'llm' for real binaries
        ./my-tests run my-feature-test --use-real-deps=git,llm

        # Swap all available mocks for their real counterparts
        ./my-tests run my-feature-test --use-real-deps=all
```

### Interactive and Debug Modes

Tend offers powerful interactive modes for debugging. The `-i` flag pauses before each step, while the `-d` flag enables a full debug environment with tmux integration, verbose logging, and disabled cleanup.

```bash
# Run interactively, stepping through each action
        ./my-tests run -i webapp-smoke-test

        # Run in debug mode. This implies -i, --no-cleanup, --tmux-split, and --very-verbose.
        # It will split your tmux window and cd into the test's temporary directory.
        ./my-tests run -d webapp-smoke-test
```

## Best Practices

### Use Helper Packages

Leverage the built-in helper packages (`fs`, `git`, `command`, `assert`, `wait`) for robust and readable tests. Avoid using raw `os` or `exec` package calls directly in steps.

### Keep Scenarios Self-Contained

Each scenario should perform its own setup and not depend on the state left by other scenarios. The harness provides an isolated temporary directory for each run to facilitate this.

### Organize with Tags

Use tags to categorize scenarios (e.g., "smoke", "api", "git", "slow"). This makes it easy to run specific subsets of tests in different contexts, such as CI or local development.

### Use Step Builders

Prefer using step builders like `harness.NewStep`, `harness.DelayStep`, and `harness.ConditionalStep` to create clear and maintainable steps.

### Leverage the Context

Use the `harness.Context` to pass state between steps and manage test artifacts. Avoid global variables to ensure scenarios are independent and can be run in parallel.

### Compile Mocks in Go

Write your mocks as Go programs instead of shell scripts. This makes them more powerful, stateful, and easier to maintain and debug. Use the `harness.SetupMocks` step to integrate them.

