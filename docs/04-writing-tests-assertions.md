The `tend` framework provides two assertion styles for different testing use cases: fail-fast hard assertions and failure-collecting soft assertions.

### 1. Hard Assertions (`ctx.Check`)

Use `ctx.Check()` for critical, fail-fast assertions. If a check fails, the current step stops immediately and is marked as failed. This style is appropriate for preconditions where subsequent steps in the scenario depend on the assertion passing.

Common use cases include:
*   Verifying critical preconditions before continuing.
*   Asserting file or resource existence before attempting to use them.
*   Validating steps in a sequence where each step depends on the previous one.

**Example:**
The following step ensures a configuration file exists before proceeding. If the check fails, the step halts.

```go
harness.NewStep("Read Configuration", func(ctx *harness.Context) error {
    configPath := filepath.Join(ctx.RootDir, "config.yml")

    // Critical check: can't continue if this fails
    if err := ctx.Check("config file exists", fs.AssertExists(configPath)); err != nil {
        return err
    }

    // Use the config file in subsequent operations...
    // ...
    return nil
}),
```

### 2. Soft Assertions (`ctx.Verify`)

Use `ctx.Verify()` to validate multiple independent properties of a single state. This style collects all assertion failures within its function block and reports them as a single aggregated error at the end. This allows you to see all problems in one test run instead of fixing them one by one.

Common use cases include:
*   Verifying multiple independent fields in command output.
*   Checking several properties of a generated file.
*   Validating multiple environment variables.

**Example:**
This step runs a command and verifies that its output contains several distinct substrings.

```go
harness.NewStep("Verify command output", func(ctx *harness.Context) error {
    result := ctx.Command("my-app", "status").Run()
    if result.Error != nil {
        return result.Error
    }

    return ctx.Verify(func(v *verify.Collector) {
        v.Contains("shows app version", result.Stdout, "App Version:")
        v.Contains("shows build info", result.Stdout, "Build:")
        v.Contains("shows status as running", result.Stdout, "Status: running")
    })
}),
```

### 3. Writing Good Assertion Descriptions

The first argument to `ctx.Check()` and to assertion methods inside `ctx.Verify()` is a description string. This description provides context in test reports and is critical for debugging failures.

A good description should be:
*   **Clear**: State what is being verified.
*   **Specific**: Include relevant context about the check.
*   **Action-oriented**: Use the present tense (e.g., "shows X", "contains Y").

**Examples:**

**Good:**
```go
// Clear and specific
v.Contains("git mock output contains branch name", result.Stdout, "On branch main")
ctx.Check("TUI is ready with save prompt", session.WaitForText("Press 's'"))
```

**Avoid:**
```go
// Too vague
v.Contains("check", result.Stdout, "text")

// Not descriptive
ctx.Check("error", err)
```