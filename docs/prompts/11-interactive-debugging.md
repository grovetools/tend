# Instructions for: Interactive Debugging

Explain `tend`'s features for interactive debugging. Base this heavily on the `tui-explorer.md` agent prompt. Adhere to the style guide.

**Content Outline:**

1.  **Overview of Debugging Features:**
    *   Explain that `tend` provides tools to pause execution and interact with the test environment.
    *   Mention that interactive debugging is built on tmux sessions, providing programmatic and manual control.

2.  **Debug Sessions (`tend run --debug-session`):**
    *   Describe `--debug-session` as the primary debugging mode that creates a dedicated, multi-window tmux session.
    *   Explain that it enables interactive mode and disables cleanup automatically.
    *   List and describe the purpose of each window in the debug session:
        *   **`runner`**: Shows the test runner's step-by-step progress. You control execution here (press Enter to advance steps).
        *   **`editor_test_dir`**: Neovim editor opened in the test's temporary root directory (the sandboxed filesystem).
        *   **`editor_test_steps`**: Neovim editor opened to the scenario's source file. Automatically jumps to the current step as you advance through the test.
        *   **`term`**: Interactive shell with the exact sandboxed environment (PATH, HOME, XDG variables) that the test uses. Critical for exploring the state of the system under test.
        *   **`logs`**: Live view of grove-core logs (if applicable).
    *   Explain the difference between the sandboxed `term` (test environment) and the user-env `editor` windows (your normal filesystem).
    *   Mention the `--server` flag to control where the debug session runs (`main` tmux server vs. `dedicated` server).

3.  **Partial Execution (`--run-steps`):**
    *   Describe the `--run-steps` flag for controlled debugging workflows.
    *   Explain that `--run-steps=1,2,3` will auto-execute Setup plus test steps 1-3, then pause at step 4 in interactive mode.
    *   State that this is useful for quickly getting to a specific point in a test without manually stepping through earlier steps.

4.  **Exploring with `tend sessions`:**
    *   Explain the `tend sessions` subcommand group for programmatically interacting with a debug session.
    *   List and explain the key commands:
        *   `sessions list`: To see active sessions.
        *   `sessions capture`: To view the screen of a pane. Mention `--wait-for`.
        *   `sessions send-keys`: To send keystrokes.
        *   `sessions kill`: To clean up.

5.  **Recording with `tend record`:**
    *   Explain the `tend record -- <command>` functionality.
    *   Describe its purpose: to capture a TUI interaction and generate reports in multiple formats (HTML, Markdown, XML).
    *   Mention that this is useful for creating documentation or providing context to an LLM for writing a test.
