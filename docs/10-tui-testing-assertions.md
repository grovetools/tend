The `grove-tend` framework provides a dedicated API for interacting with and asserting the state of Terminal User Interface (TUI) applications running within a test session. These functions allow tests to read screen content, wait for specific text to appear, and verify UI state in a reliable manner.

### 1. Capturing Screen Content (`session.Capture`)

The `session.Capture` method is the fundamental tool for reading the current state of the TUI. It returns the visible text content of the application's terminal pane.

By default, `Capture` returns the raw output, including ANSI escape codes used for color and styling. For text-based assertions, it is often more effective to use the `WithCleanedOutput()` option to get a plain text representation of the screen.

```go
import "github.com/mattsolo1/grove-tend/pkg/tui"

// ... inside a test step

session := ctx.Get("tui_session").(*tui.Session)

// Capture raw output with ANSI codes
rawContent, err := session.Capture(tui.WithRawOutput())

// Capture plain text content (recommended for assertions)
cleanContent, err := session.Capture(tui.WithCleanedOutput())
```

### 2. Waiting for Content (`session.WaitForText`)

Because TUIs often update asynchronously (e.g., loading data, running animations), the most reliable way to assert state is to wait for expected content to appear. The `session.WaitForText` method polls the TUI screen until a specified string is found or a timeout is reached.

This is the primary method for verifying that a UI has reached an expected state after an action.

**Example from `tests/e2e/scenarios_tui.go`:**
This step launches the TUI and waits for the main title to appear before proceeding.

```go
harness.NewStep("Launch TUI and verify initial state", func(ctx *harness.Context) error {
    // ... setup code
    session, err := ctx.StartTUI(tendBinary, []string{"tui"})
    if err != nil {
        return fmt.Errorf("failed to start `tend tui`: %w", err)
    }
    ctx.Set("tui_session", session)

    // Wait for the TUI to load by looking for the header text
    if err := session.WaitForText("Tend Test Runner", 10*time.Second); err != nil {
        content, _ := session.Capture()
        return fmt.Errorf("TUI did not load: %w\nContent:\n%s", err, content)
    }
    // ...
})
```

### 3. Immediate Assertions (`session.AssertContains`)

The `session.AssertContains` and `session.AssertNotContains` methods perform an immediate check on the current screen content without polling. They are best used inside `ctx.Check` (for fail-fast assertions) or `ctx.Verify` (for collecting multiple assertions).

These methods are useful for verifying a state that is known to be stable.

**Example using `ctx.Check`:**
This is the standard way to perform a critical, fail-fast check.

```go
// ... inside a test step

// Check for a save confirmation message
err := ctx.Check("file save confirmation appears",
    session.WaitForText("File saved to output.txt", 2*time.Second))
if err != nil {
    return err
}
```

**Example using `ctx.Verify`:**
To use these methods in a `ctx.Verify` block, check that the returned error is `nil`.

```go
// ... inside a test step

// Verify that multiple projects are visible on the initial screen
return ctx.Verify(func(v *verify.Collector) {
    v.Equal("project-a is visible", nil, session.AssertContains("project-a"))
    v.Equal("project-b is visible", nil, session.AssertContains("project-b"))
})
```

### 4. Advanced Navigation and Assertions

For more complex TUI interactions, the framework provides higher-level assertion and navigation helpers.

*   **`session.NavigateToText(text)`**: This high-level action moves the TUI's cursor or selection to the line containing the specified `text`. It automatically calculates the required number of "Up" or "Down" key presses.

*   **`session.AssertLine(predicate, message)`**: This method iterates through each line of the TUI's visible screen content and passes if the provided predicate function returns `true` for any line. This is useful for checking line-specific states, such as identifying the currently selected item by its cursor.

**Example from `tests/e2e/scenarios_tui.go`:**
This step navigates to a specific line and then uses `AssertLine` to verify that the selection cursor (`>`) is on the correct line.

```go
harness.NewStep("Navigate to docs/guide.md using NavigateToText", func(ctx *harness.Context) error {
    session := ctx.Get("advanced_session").(*tui.Session)

    // Navigate directly to the line containing "docs/guide.md"
    if err := session.NavigateToText("docs/guide.md"); err != nil {
        return fmt.Errorf("failed to navigate: %w", err)
    }

    // Verify the selection indicator moved to the correct line
    return session.AssertLine(func(line string) bool {
        return strings.Contains(line, "> docs/guide.md")
    }, "expected '> docs/guide.md' to be selected")
}),
```