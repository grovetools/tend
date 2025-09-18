## Usage Patterns

### Basic File Operations

The `fs` helper package provides a convenient way to perform common file system operations within a temporary, isolated test directory managed by the harness context.

```go
harness.NewStep("Setup test project structure", func(ctx *harness.Context) error {
    // Create a file with content in the scenario's temporary root directory
    if err := fs.WriteString(filepath.Join(ctx.RootDir, "main.go"), "package main"); err != nil {
        return err
    }
    // Create a subdirectory
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
    // Use ctx.Command to respect the mock-aware PATH
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

### Controlling Scenario Execution

Use `LocalOnly` and `ExplicitOnly` properties in your `Scenario` definition to control when tests run. `LocalOnly` scenarios are skipped in CI unless `--include-local` is used. `ExplicitOnly` scenarios are skipped during `run all` and must be invoked by name or with the `--explicit` flag, useful for expensive or destructive tests.

```go
var LocalOnlyScenario = &harness.Scenario{
    Name:        "local-dev-setup",
    Description: "Should only run on developer machines, not in CI.",
    LocalOnly:   true, // Skips in CI by default
    Steps: []harness.Step{ /* ... */ },
}

var ExplicitOnlyScenario = &harness.Scenario{
    Name:         "expensive-integration-test",
    Description:  "Must be run explicitly by name, not with 'tend run'.",
    ExplicitOnly: true, // Requires 'tend run expensive-integration-test'
    Steps: []harness.Step{ /* ... */ },
}
```

### Managing Background Processes

For scenarios that require a long-running background process, such as a server, use `cmd.Start()` to get a `Process` handle. This allows your test steps to continue while the process runs, and you can manage its lifecycle with `process.Wait()` or `process.Kill()`.

```go
harness.NewStep("Start and manage a background server", func(ctx *harness.Context) error {
    // Start a server process in the background
    serverCmd := ctx.Command("./my-server", "--port", "8080")
    process, err := serverCmd.Start()
    if err != nil {
        return fmt.Errorf("failed to start server: %w", err)
    }

    // Give the server a moment to start up
    time.Sleep(500 * time.Millisecond)

    // ... perform other test steps like making API calls ...

    // Stop the server at the end of the step
    if err := process.Kill(); err != nil {
        return fmt.Errorf("failed to kill server: %w", err)
    }

    result := process.Wait(2 * time.Second) // Wait for cleanup
    return nil
}),
```

