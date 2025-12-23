## TUI Session Management

`tend` tests Terminal User Interface (TUI) applications by running them in isolated `tmux` sessions. This allows the test harness to programmatically observe the TUI's state and send keystrokes to it, simulating user interaction.

### 1. Starting a TUI Session (`ctx.StartTUI`)

The primary function for launching a TUI application is `ctx.StartTUI`. It takes the path to a binary and its arguments, launches it in a new, isolated `tmux` session, and returns a `*tui.Session` handle for interaction.

The harness automatically manages the lifecycle of this `tmux` session, ensuring it is created before the test step and terminated during cleanup.

```go
harness.NewStep("Launch the TUI", func(ctx *harness.Context) error {
    // Path to the TUI binary to be tested
    // This example assumes a fixture binary has been built
    binPath := "./fixtures/bin/my-tui-app"

    // Launch the TUI in a new session
    session, err := ctx.StartTUI(binPath, []string{})
    if err != nil {
        return err
    }

    // Store the session handle in the context for subsequent steps
    ctx.Set("tui_session", session)

    // Wait for the TUI to render its initial view
    return session.WaitForText("Main Menu", 5*time.Second)
}),
```

#### Configuration Options

You can configure the TUI's execution environment using options:

*   **`tui.WithEnv(vars ...string)`**: Sets environment variables for the TUI process.
*   **`tui.WithCwd(path string)`**: Sets the working directory for the TUI process.

```go
// Launch TUI with a specific configuration and working directory
session, err := ctx.StartTUI(
    binPath,
    []string{"--config", "test.yml"},
    tui.WithEnv("LOG_LEVEL=debug"),
    tui.WithCwd(ctx.Dir("test-data")),
)
```

### 2. The `tui.Session` Object

The `ctx.StartTUI` function returns a `*tui.Session` object, which is the handle for interacting with the running TUI. It provides methods for:

*   Sending keystrokes (`SendKeys`, `Type`)
*   Capturing screen content (`Capture`)
*   Waiting for specific states (`WaitForText`, `WaitStable`)
*   Asserting on screen content (`AssertContains`, `AssertLine`)

This object is typically stored in the test `Context` to be used across multiple steps.

### 3. Headless Testing (`ctx.StartHeadless`)

For testing `bubbletea` applications, `tend` provides a headless testing mode via `ctx.StartHeadless`. This function runs a `bubbletea` model's `Update` and `View` logic directly, without launching a real terminal or `tmux` session.

This approach is faster and useful for unit-testing a model's state transitions and view output without the overhead of a full TUI environment.

```go
harness.NewStep("Test model logic headlessly", func(ctx *harness.Context) error {
    // myapp.InitialModel() returns a tea.Model
    model := myapp.InitialModel()
    
    // Start a headless session
    session := ctx.StartHeadless(model)
    
    // Interact with the model by sending messages
    session.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
    
    // Assert on the model's view output
    output := session.Output()
    if !strings.Contains(output, "Selected Item: 2") {
        return fmt.Errorf("selection did not update correctly")
    }
    
    return nil
}),
```