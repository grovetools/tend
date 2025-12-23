## Executing External Commands (`ctx.Command`)

To run external commands within a test step, use the `ctx.Command` method. This method is the standard way to execute any external process, such as `git`, `docker`, or other CLIs.

Its primary function is to ensure test isolation. It automatically prepends the test's temporary `bin` directory (created by `harness.SetupMocks`) to the `PATH`. This guarantees that any configured mock binaries are executed instead of their real system counterparts. Additionally, it sandboxes environment variables like `HOME` and `XDG_*` to prevent tests from affecting the user's local configuration.

After creating a command, execute it with `.Run()` to get a `*command.Result` object, which contains the command's `Stdout`, `Stderr`, `ExitCode`, and any execution `Error`.

```go
harness.NewStep("Run mocked git command", func(ctx *harness.Context) error {
    // ctx.Command ensures that 'git' resolves to the mock binary
    // configured in a previous SetupMocks step.
    cmd := ctx.Command("git", "status")
    result := cmd.Run()

    // The result object contains stdout, stderr, exit code, and any execution error.
    if err := result.AssertSuccess(); err != nil {
        return err
    }

    // Verify the output came from the mock.
    return ctx.Check("mock git status is correct",
        assert.Contains(result.Stdout, "On branch main"),
    )
}),
```

## Testing the Project Binary (`ctx.Bin`)

For executing the main binary of the project under test, `ctx.Bin` serves as a specialized helper. It offers a more direct way to run project-specific subcommands without needing to manually track the binary's path.

This function discovers the binary's location by reading the `binary.path` field from the project's `grove.yml` file. It then constructs a command within the same sandboxed environment as `ctx.Command`.

Instead of manually referencing the binary path:
```go
// This requires knowing the path to the project binary.
cmd := ctx.Command(ctx.GroveBinary, "plan", "init", "my-plan")
```

Use `ctx.Bin` for a more direct approach:
```go
// ctx.Bin automatically finds the binary path from grove.yml.
cmd := ctx.Bin("plan", "init", "my-plan")
```