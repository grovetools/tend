# Instructions for: TUI Testing: Session Management

Explain how to start and manage a TUI testing session. Adhere to the style guide.

**Content Outline:**

1.  **Overview of TUI Testing:**
    *   Explain that `tend` uses `tmux` to run TUI applications in isolated sessions, allowing for programmatic control and observation.

2.  **Starting a TUI Session (`ctx.StartTUI`):**
    *   Explain the `ctx.StartTUI` function.
    *   Show an example of launching a TUI binary and getting a `*tui.Session` handle.
    *   Mention the available options, like `tui.WithEnv` and `tui.WithCwd`.

3.  **The `tui.Session` Object:**
    *   Describe the `Session` object as the handle for interacting with the running TUI application.

4.  **Headless Testing (`ctx.StartHeadless`):**
    *   Briefly describe `ctx.StartHeadless` as an alternative for testing Bubble Tea model logic without the overhead of a real terminal.
