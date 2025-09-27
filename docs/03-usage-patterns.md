An expert Go developer and technical writer would produce the following documentation for `docs/usage-patterns.md`.

This updated version incorporates new features found in the context, such as advanced TUI testing and mocking capabilities, while also restructuring and clarifying existing patterns for better developer experience. It directly addresses the user's prompt by providing clear, practical examples for the requested usage patterns.

```markdown
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

The `command` package simplifies running external commands. Using `ctx.Command()` is crucial as it ensures that any configured mocks are respected by searching a temporary `bin` directory first. The result includes stdout, stderr, exit code, and execution duration.

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

`tend` provides a first-class mocking system where mocks are defined as Go binaries. The `harness.SetupMocks` step builder creates a sandboxed `PATH` where your mock binaries are used instead of the real ones.

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

For integration testing, you can selectively swap mocks for their real counterparts using the `--use-real-deps` flag. `tend` uses `grove dev current <tool>` to discover the path to the active binary in your ecosystem.

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

### Testing Terminal User Interfaces (TUIs)

`tend` provides an effective way to test interactive TUIs by automating `tmux` sessions.

- **For any TUI**: Use `ctx.StartTUI` to launch the application in an isolated `tmux` session.
- **For BubbleTea apps**: Use `ctx.StartHeadless` to test model logic without a terminal, which is faster and ideal for unit/integration tests.

```go
// Example of testing a TUI in tmux
harness.NewStep("Test help command", func(ctx *harness.Context) error {
    // Launch the TUI in a managed tmux session
    session, err := ctx.StartTUI("./my-tui-app")
    if err != nil {
        return err
    }
    
    // Wait for the UI to be ready
    if err := session.WaitForText("Welcome", 5*time.Second); err != nil {
        return err
    }

    // Interact with the TUI
    if err := session.SendKeys("h"); err != nil { // Send 'h' for help
        return err
    }

    // Assert on the output
    return session.WaitForText("Help content appears here", 2*time.Second)
}),
```

### Interactive and Debug Modes

`tend` offers interactive modes for debugging. The `-i` flag pauses before each step, while the `-d` flag enables a full debug environment with `tmux` integration, verbose logging, and disabled cleanup.

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
// Example from grove-flow testing an orchestrator process
harness.NewStep("Run plan in background and verify polling starts", func(ctx *harness.Context) error {
    flow, _ := getFlowBinary()

    // Run the plan orchestrator as a background process
    cmd := ctx.Command(flow, "plan", "run", "polling-plan", "--all").Dir(ctx.RootDir)
    process, err := cmd.Start()
    if err != nil {
        return fmt.Errorf("failed to start plan run: %v", err)
    }
    ctx.Set("polling_process", process)

    // Give it a moment to launch and check its output for expected state
    time.Sleep(2 * time.Second)
    stdout := process.Stdout()
    if !strings.Contains(stdout, "flow plan complete") {
        return fmt.Errorf("expected launch message not found: %s", stdout)
    }
    
    // ... other steps can now run ...

    // At the end, you can wait for the process to finish or kill it.
    // process.Wait(30 * time.Second)
    return nil
}),
```
```