# Grove Tend Testing Library - Comprehensive Guide

# Grove Tend: A Go Library for E2E Testing

A Go library for creating powerful, scenario-based end-to-end testing frameworks. Grove Tend provides the essential building blocks to replace fragile, ad-hoc bash scripts with structured, maintainable, and easily debuggable Go code.

Designed with a **library-first philosophy**, Grove Tend empowers you to build a custom testing CLI tailored specifically to your project's needs. This approach keeps your test definitions and logic directly within your Go codebase, improving discoverability and maintainability.

The framework offers a comprehensive suite of features to streamline the entire E2E testing lifecycle:

- **Scenario-Based Structure:** Organize tests logically with `Scenarios`, `Steps`, and a shared `Context`.
- **First-Class Mocking:** Define mocks in Go, compile them as binaries, and seamlessly swap between mocked and real dependencies during test runs.
- **Interactive Debugging:** Step through complex scenarios one-by-one or leverage the powerful debug mode with automatic tmux integration for an unparalleled debugging experience.
- **Rich Helper Packages:** Utilize built-in helpers for filesystem, Git, command execution, and assertions to write robust tests quickly.
- **CI/CD Integration:** Generate standard JUnit or JSON reports and benefit from automatic GitHub Actions annotations for seamless integration into your pipelines.

By combining the power of Go with a thoughtfully designed testing harness, Grove Tend transforms end-to-end testing from a chore into a core part of your development workflow.

## Core Concepts

### Scenario

A Scenario is the fundamental unit of a test in Grove Tend. It represents a complete end-to-end test case, composed of a series of logical steps. Each scenario has a name, description, and tags, which help in organizing, filtering, and understanding the purpose of the test.

```go
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

A Step is a single, atomic action within a Scenario. Each step has a name and a function that contains the test logic. Steps are executed sequentially, and if any step fails, the scenario stops. The framework provides step builders like `harness.NewStep` for convenience.

```go
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
harness.NewStep("Step 1: Create a file", func(ctx *harness.Context) error {
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
```

### Harness

The Harness is the engine that orchestrates scenario execution. While developers primarily interact with `Scenario`, `Step`, and `Context`, the `Harness` is responsible for setting up the test environment, running steps sequentially, handling errors, and performing cleanup. It's typically invoked via `app.Execute` in the test runner's `main` function.

```go
func main() {
    // Collect all scenarios for your test runner
    scenarios := []*harness.Scenario{
        MyWebAppScenario,
        // Add more scenarios here...
    }

    // Execute the tend application with your scenarios
    if err := app.Execute(context.Background(), scenarios); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

### Tags

Tags are labels used to categorize Scenarios. They provide a powerful mechanism for selectively running groups of tests, such as running only `smoke` tests in a CI pipeline or only tests related to `git` during development. They are specified as a slice of strings in the Scenario definition and used with the `--tags` command-line flag.

```go
var GitIntegrationScenario = &harness.Scenario{
    Name:        "example-git-integration",
    Description: "Tests core Git functionality.",
    Tags:        []string{"git", "integration", "smoke"},
    Steps:       []harness.Step{ /* ... */ },
}

// To run this scenario, you could use:
// ./my-tests run --tags=smoke
```

### Managed Resources

For each scenario run, the Harness creates an isolated set of resources, primarily a temporary root directory. This ensures tests are self-contained and do not interfere with each other. The `Context` provides access to this managed directory, where tests can safely create files, initialize repositories, or store artifacts, all of which are automatically cleaned up after the test completes (unless `--no-cleanup` is used).

```go
harness.NewStep("Work with isolated resources", func(ctx *harness.Context) error {
    // ctx.RootDir is the unique, temporary root for this scenario run
    projectDir := ctx.NewDir("my-project")
    
    // All operations within projectDir are isolated and cleaned up
    err := fs.WriteString(filepath.Join(projectDir, "config.json"), "{}")
    if err != nil {
        return err
    }
    
    // The directory is automatically removed after the scenario
    return nil
})
```

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

## Best Practices

### Use Descriptive and Consistent Naming

Scenario and Step names should be clear, concise, and descriptive. A good name immediately communicates the test's intent and makes failures easier to diagnose from logs and reports.

- **Scenario Names**: Use a consistent pattern like `feature-or-entity-action`. For example, `git-workflow` or `webapp-smoke-test`.
- **Step Names**: Describe the action being performed, such as `Initialize git repository` or `Verify file exists`.
- **Descriptions**: Use the `Description` field on both Scenarios and Steps to provide additional context that appears in the `tend list --verbose` output.

### Embrace Structured Error Handling

Every `Step.Func` should handle errors properly by returning them. The harness will automatically stop the scenario on the first non-nil error.

- **Wrap Errors**: Provide context when returning errors to make debugging easier. Use `fmt.Errorf` with the `%w` verb: `return fmt.Errorf("failed to write config: %w", err)`.
- **Use Assertions**: Leverage the `pkg/assert` package for readable and explicit checks. `return assert.Contains(result.Stdout, "expected output")` is clearer than manual string checks and provides better failure messages.

### Ensure Scenarios are Self-Contained and Isolated

Each scenario should be able to run independently without relying on the state left by others. The harness facilitates this by providing a unique, temporary directory for each scenario run, accessible via `ctx.RootDir`.

- **Use the Test Directory**: All test artifacts, files, and repositories should be created within `ctx.RootDir`.
- **Manage State with Context**: Pass data between steps using `ctx.Set()` and `ctx.Get()`. Avoid using global variables, which can create dependencies between tests and prevent parallel execution.

### Write Efficient and Performant Tests

Avoid fixed-length sleeps, as they make tests slow and flaky. Instead of `time.Sleep()` or `harness.DelayStep`, use polling mechanisms that wait for a specific condition to be met.

- **Use Wait Helpers**: The `pkg/wait` package provides functions like `wait.For`, `wait.ForHTTP`, and `wait.ForFileContent` that poll a condition until it succeeds or a timeout is reached. This makes tests both faster and more reliable.

### Structure Scenarios for Maintainability

Well-structured tests are easier to read, update, and debug.

- **Leverage Helper Packages**: Use the built-in helpers (`fs`, `git`, `command`, `assert`, `wait`) to perform common operations robustly.
- **Use Step Builders**: Use functions like `harness.NewStep` and `harness.ConditionalStep` to create clear, reusable steps.
- **Write Mocks in Go**: Instead of brittle shell scripts, write your mocks as Go programs. This makes them more powerful, stateful, and easier to maintain. Integrate them using the `harness.SetupMocks` step.

### Design Scenarios for CI/CD

Tend provides features to help manage which tests run in different environments.

- **Local-Only Scenarios**: Mark scenarios that require a specific local setup or are not suitable for CI with `LocalOnly: true`. They will be automatically skipped in CI environments.
- **Explicit-Only Scenarios**: For expensive or long-running integration tests, use `ExplicitOnly: true`. These tests are skipped by default during a full `tend run` and must be invoked by name or with the `--explicit` flag.
- **Generate Reports**: Use the `--junit <file>` and `--json <file>` flags in your CI pipeline to generate machine-readable reports for test analytics and integration with other tools.

### Master Debugging Techniques

Tend offers powerful tools to troubleshoot failing tests.

- **Interactive Mode (`-i`):** Pause execution before each step, allowing you to inspect the state of the system.
- **No Cleanup (`--no-cleanup`):** Prevents the deletion of the temporary test directory (`ctx.RootDir`), so you can examine the generated files and logs after a run.
- **Verbose Output (`-v`, `--very-verbose`):** Increase the level of detail in the output. `--very-verbose` includes command output.
- **Debug Mode (`-d`):** The ultimate debugging tool. It's a shorthand for `-i --no-cleanup --very-verbose --tmux-split`, which runs the test interactively and automatically splits your `tmux` window, `cd`-ing into the test's temporary directory for hands-on inspection.

