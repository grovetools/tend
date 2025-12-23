# Instructions for: TUI Testing: Assertions

Explain the APIs for asserting the state of a running TUI. Reference `tests/e2e/scenarios_tui.go`. Adhere to the style guide.

**Content Outline:**

1.  **Capturing Screen Content (`session.Capture`):**
    *   Explain that `session.Capture` returns the text content of the TUI pane.
    *   Mention the `WithCleanedOutput()` option to strip ANSI codes for easier text matching.

2.  **Waiting for Content (`session.WaitForText`):**
    *   Describe `session.WaitForText` as the primary method for asserting that specific content has appeared on screen. Explain that it polls until the text is found or a timeout occurs.

3.  **Immediate Assertions (`session.AssertContains`):**
    *   Explain `session.AssertContains` and `session.AssertNotContains` for immediate checks.
    *   Show how to use these inside `ctx.Check` or `ctx.Verify`.

4.  **Advanced Navigation and Assertions:**
    *   Briefly describe `session.NavigateToText` as a higher-level action to move a cursor or selection to a line containing specific text.
    *   Briefly describe `session.AssertLine` as a way to check for conditions on a line-by-line basis (e.g., finding the line with a selection cursor `>`).
