# Grove Tend Testing Library - Comprehensive Guide

Grove Tend is a Go library for creating powerful, scenario-based end-to-end testing frameworks. It provides the building blocks to replace ad-hoc bash scripts with structured, maintainable, and debuggable Go code. Designed as a pure library, it allows developers to build a custom testing CLI tailored to their project's specific needs, keeping test definitions and logic close to the application code. Its core philosophy centers on structured testing, rich helper packages for common operations, and first-class support for mocking and debugging.

## Core Concepts

### Scenario

A Scenario is the fundamental unit of a test in grove-tend. It represents a complete end-to-end test case, logically grouping a series of actions (Steps). Each Scenario has a name for identification, a description of its purpose, and tags for organization and filtering, allowing you to run specific subsets of your test suite.

```go
// A minimal Scenario definition.
var MyWebAppScenario = &harness.Scenario{
    Name:        "webapp-smoke-test",
    Description: "Performs a basic smoke test on the web application.",
    Tags:        []string{"smoke", "webapp"},
    Steps: []harness.Step{
        // ... Steps go here ...
    },
}
```

### Step

A Step is a single, atomic action within a Scenario. It consists of a name and a function that performs the test logic. Steps are executed sequentially, and if any step returns an error, the Scenario fails and stops. This structure breaks down complex tests into manageable, readable, and debuggable parts.

```go
var MyWebAppScenario = &harness.Scenario{
    Name: "webapp-smoke-test",
    Steps: []harness.Step{
        // A step can be defined using a struct literal.
        {
            Name: "Create a test file",
            Func: func(ctx *harness.Context) error {
                filePath := filepath.Join(ctx.RootDir, "config.yml")
                return fs.WriteString(filePath, "setting: true")
            },
        },
        // Or using the NewStep helper for cleaner syntax.
        harness.NewStep("Verify file was created", func(ctx *harness.Context) error {
            if !fs.Exists(filepath.Join(ctx.RootDir, "config.yml")) {
                return fmt.Errorf("config.yml was not created")
            }
            return nil
        }),
    },
}
```

### Context

The harness.Context is a state container that is passed to every Step within a single Scenario execution. Its primary purpose is to share data and state between steps. It provides a key-value store (Set/Get) and manages a temporary root directory (RootDir) for the test, ensuring that each scenario runs in a clean, isolated environment.

```go
var MyScenario = &harness.Scenario{
    Name: "context-passing-example",
    Steps: []harness.Step{
        harness.NewStep("Setup test directory", func(ctx *harness.Context) error {
            // Create a new directory within the scenario's temp space.
            testDir := ctx.NewDir("webapp-test")
            // Store its path in the context for the next step.
            ctx.Set("test_dir", testDir) 
            return fs.CreateDir(testDir)
        }),
        harness.NewStep("Run a command in the directory", func(ctx *harness.Context) error {
            // Retrieve the directory path from the context.
            testDir := ctx.GetString("test_dir") // or ctx.Dir("webapp-test")
            
            cmd := command.New("pwd").Dir(testDir)
            result := cmd.Run()
            ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
            return result.Error
        }),
    },
}
```

## Usage Patterns

### Basic File Operations

The `fs` helper package provides a convenient API for filesystem operations within a test's temporary directory. This avoids boilerplate code for creating files, checking existence, and reading content, making steps cleaner and more focused on the test logic.

```go
harness.NewStep("Work with files", func(ctx *harness.Context) error {
    // Create a directory.
    srcDir := filepath.Join(ctx.RootDir, "src")
    if err := fs.CreateDir(srcDir); err != nil {
        return err
    }

    // Write a file.
    mainGoPath := filepath.Join(srcDir, "main.go")
    if err := fs.WriteString(mainGoPath, "package main"); err != nil {
        return err
    }

    // Check if the file exists.
    if !fs.Exists(mainGoPath) {
        return fmt.Errorf("main.go should exist")
    }

    // Read the file's content.
    content, err := fs.ReadString(mainGoPath)
    if err != nil {
        return err
    }
    if content != "package main" {
        return fmt.Errorf("unexpected file content")
    }
    return nil
}),
```

### Command Execution

The `command` helper package is used to execute external shell commands. It captures stdout, stderr, exit code, and execution duration. This is essential for testing CLIs or any tool that interacts with the shell. The `ctx.ShowCommandOutput` helper can be used to display formatted command results in the test logs.

```go
harness.NewStep("Run an external command", func(ctx *harness.Context) error {
    cmd := command.New("echo", "Hello, Tend!").Dir(ctx.RootDir)
    result := cmd.Run()

    // Display the command and its output for debugging.
    ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
    
    // Check for execution errors.
    if result.Error != nil {
        return fmt.Errorf("command failed to run: %w", result.Error)
    }

    // Assert on the output.
    if !strings.Contains(result.Stdout, "Hello, Tend!") {
        return fmt.Errorf("unexpected command output: %s", result.Stdout)
    }
    return nil
}),
```

### Git Operations

The `git` helper package simplifies testing workflows that involve Git. It provides functions for common operations like initializing a repository, configuring it for tests, adding files, and committing, abstracting away the underlying shell commands.

```bash
harness.NewStep("Setup and commit to a git repo", func(ctx *harness.Context) error {
    repoDir := ctx.NewDir("my-repo")
    
    // Initialize a new repository.
    if err := git.Init(repoDir); err != nil {
        return err
    }
    // Set user.name and user.email to avoid test failures in CI.
    if err := git.SetupTestConfig(repoDir); err != nil {
        return err
    }
    
    // Create, add, and commit a file.
    fs.WriteString(filepath.Join(repoDir, "README.md"), "# My Project")
    if err := git.Add(repoDir, "README.md"); err != nil {
        return err
    }
    if err := git.Commit(repoDir, "Initial commit"); err != nil {
        return err
    }
    
    return nil
}),
```

### Mocking Dependencies

Grove Tend provides a first-class solution for mocking external CLI dependencies. Instead of brittle shell scripts, mocks are written as simple Go programs. The `harness.SetupMocks` step prepares the test environment by placing these mocks onto the PATH for the scenario. Subsequent commands should then be created with `ctx.Command()` to ensure they use the modified PATH and execute the mocks instead of real binaries.

```go
var GitWorkflowScenario = &harness.Scenario{
    Name:        "git-workflow-with-mock",
    Description: "Tests a git workflow using a mocked git command",
    Steps: []harness.Step{
        // Tell tend to find './bin/mock-git' and add it to the PATH.
        harness.SetupMocks(
            harness.Mock{CommandName: "git"},
        ),
        
        harness.NewStep("Initialize git repository", func(ctx *harness.Context) error {
            repoDir := ctx.NewDir("git-test")
            
            // ctx.Command ensures the command is found in the mocked PATH.
            cmd := ctx.Command("git", "init").Dir(repoDir)
            result := cmd.Run()
            
            ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
            return assert.Contains(result.Stdout, "Initialized empty Git repository", "should use the mock git init")
        }),
    },
}
```

### Swapping Mocks for Real Dependencies

For integration testing, it's often useful to swap a mock for its real counterpart. Grove Tend supports this via the `--use-real-deps` command-line flag. When this flag is used, `tend` will ignore the specified mock and instead find the path to the real, active binary using the Grove ecosystem's `grove dev current <tool>` command. This allows the same test scenario to be run as a pure unit test (with all mocks) or as a targeted integration test.

```bash
# Run a scenario, but swap the 'git' mock for the real binary
./my-tests run git-workflow-with-mock --use-real-deps=git

# Run a scenario with multiple dependencies, swapping two of them for real binaries
./my-tests run mixed-dependencies --use-real-deps=git,docker

# Run a scenario, swapping all available mocks for their real counterparts
./my-tests run mixed-dependencies --use-real-deps=all
```

### Interactive and Debug Modes

Grove Tend includes powerful debugging features. Interactive mode (`-i`) pauses before each step, allowing you to inspect the state of the filesystem and proceed one step at a time. Debug mode (`-d`) is even more powerful, implying interactive mode and also enabling verbose logging, preventing cleanup of the temporary test directory, and integrating with `tmux` to provide a dedicated shell within the test environment for manual inspection and command execution.

```bash
# Run a scenario interactively, pausing before each step
./my-tests run -i my-scenario

# Run a scenario in debug mode for full inspection capabilities
./my-tests run -d my-scenario
```

## Best Practices

### Project Setup

A typical project using `grove-tend` will have a `tests/e2e/` directory containing the test runner's `main.go` and scenario files. If using mocks, a `tests/mocks/` directory holds the mock source code. A `Makefile` is used to build both the main application binary and the mock binaries, placing them in a `./bin/` directory (e.g., `./bin/my-app` and `./bin/mock-git`). This convention is essential for `tend`'s mock discovery.

### Test Organization

Organize scenarios into logical files (e.g., `scenarios_basic.go`, `scenarios_git.go`). The main test runner (`main.go`) then collects all scenario definitions into a single slice passed to `app.Execute`. Use tags (e.g., "smoke", "git", "regression") to categorize scenarios. This allows you to run specific test suites easily from the command line (e.g., `tend run --tags=smoke`).

### Writing Scenarios

Keep scenarios focused on a single workflow or feature. Break down complex logic into small, descriptively named steps. Use the `harness.Context` to pass only necessary data between steps, such as file paths or identifiers, rather than large, complex objects. Leverage the helper packages (`fs`, `git`, `command`, `assert`) to keep test code concise and readable.

