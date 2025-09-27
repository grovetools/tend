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