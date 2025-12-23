## Filesystem Operations

Each test scenario runs in an isolated, temporary root directory created by the harness. This environment is automatically cleaned up after the test completes, unless the `--no-cleanup` flag is used.

### Sandboxed Environment

The harness automatically provisions a sandboxed home directory structure within the test's root directory. This structure mimics a standard user home environment:

```
/tmp/tend-scenario-XXXX/    (ctx.RootDir)
└── home/                   (ctx.HomeDir())
    ├── .config/            (ctx.ConfigDir())
    ├── .local/
    │   └── share/          (ctx.DataDir())
    └── .cache/             (ctx.CacheDir())
```

For any command executed via `ctx.Command()` or `ctx.Bin()`, the following environment variables are automatically set to point to these sandboxed locations:
*   `HOME`
*   `XDG_CONFIG_HOME`
*   `XDG_DATA_HOME`
*   `XDG_CACHE_HOME`

This mechanism ensures that applications under test interact with the isolated test environment instead of the user's actual home directory.

### Creating Directories

Use `ctx.NewDir(name)` to create a directory within the test's root directory. The function returns the absolute path to the newly created directory.

```go
// Creates a directory at /tmp/tend-scenario-XXXX/my-workspace
workspaceDir := ctx.NewDir("my-workspace")
```

### Accessing Sandboxed Paths

The `harness.Context` provides properties to access key paths within the sandboxed environment:

*   **`ctx.RootDir`**: The absolute path to the root temporary directory for the scenario.
*   **`ctx.HomeDir()`**: The path to the sandboxed `home/` directory.
*   **`ctx.ConfigDir()`**: The path to the sandboxed `home/.config` directory.
*   **`ctx.DataDir()`**: The path to the sandboxed `home/.local/share` directory.
*   **`ctx.CacheDir()`**: The path to the sandboxed `home/.cache` directory.

### Filesystem Helpers (`pkg/fs`)

The `pkg/fs` package provides helpers for common filesystem operations and assertions, which simplifies file manipulation and verification.

**Example: Creating and Verifying a File**

The following example demonstrates creating a directory, writing a file, and asserting its existence.

```go
harness.NewStep("Create and verify file", func(ctx *harness.Context) error {
    // 1. Create a directory for a test project.
    projectDir := ctx.NewDir("my-project")

    // 2. Write a configuration file into the new directory.
    // fs.WriteString handles creating parent directories if needed.
    configPath := filepath.Join(projectDir, "config.yml")
    configContent := "setting: value"
    if err := fs.WriteString(configPath, configContent); err != nil {
        return fmt.Errorf("failed to write config file: %w", err)
    }

    // 3. Verify the file was created using a filesystem assertion.
    // ctx.Check provides a hard, fail-fast assertion.
    if err := ctx.Check("config file exists", fs.AssertExists(configPath)); err != nil {
        return err
    }

    return nil
}),
```