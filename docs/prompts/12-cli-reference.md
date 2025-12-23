# Instructions for: CLI Reference

Provide a concise reference for the main `tend` CLI commands and flags. Adhere to the style guide.

**Content Outline:**

For each of the following commands, provide a one-sentence description and list its most important flags.

1.  **`tend run [scenario...]`**
    *   Description: Executes test scenarios with support for filtering, parallelization, and interactive debugging.
    *   Important flags:
        *   `--parallel` / `-p`: Run scenarios in parallel.
        *   `--tags` / `-t`: Filter scenarios by tags.
        *   `--interactive` / `-i`: Enable step-through interactive mode.
        *   `--debug-session`: Start a multi-window tmux debug session.
        *   `--no-cleanup`: Keep temporary directories after test completion.
        *   `--use-real-deps`: Swap specific mocks with real binaries (e.g., `--use-real-deps=flow` or `--use-real-deps=all`).
        *   `--run-steps`: Auto-execute specific test steps then pause (e.g., `--run-steps=1,2,3`).
        *   `--explicit`: Run only scenarios marked as `ExplicitOnly`.
        *   `--format`: Output format (`text`, `json`, `junit`).
        *   `--junit`: Write JUnit XML report to file.
        *   `--record-tui`: Directory to save TUI recordings for failed tests.

2.  **`tend list`**
    *   Description: Lists all available test scenarios with optional filtering.
    *   Important flags:
        *   `--tags`: Filter scenarios by tags.
        *   `--keyword`: Filter scenarios by keyword in name or description.
        *   `--verbose` / `-v`: Show detailed scenario information.

3.  **`tend tui`**
    *   Description: Launches an interactive terminal UI for browsing and running test scenarios.
    *   No major flags to document.

4.  **`tend validate`**
    *   Description: Validates test scenario definitions without executing them.
    *   Checks for structural issues, missing steps, or configuration errors.

5.  **`tend ecosystem run`**
    *   Description: Discovers and runs tests across all projects in a Grove ecosystem workspace.
    *   Executes conventional `make` targets (e.g., `test-e2e`) in parallel across all discovered projects.
    *   Aggregates results into a summary report.

6.  **`tend sessions <subcommand>`**
    *   Description: Programmatically interact with tmux-based test sessions.
    *   Subcommands:
        *   `list`: Show active debug sessions.
        *   `capture`: Capture the screen content of a pane (supports `--wait-for` to wait for specific text).
        *   `send-keys`: Send keystrokes to a session.
        *   `kill`: Terminate a debug session.
        *   `attach`: Attach to a running session.

7.  **`tend record -- <command>`**
    *   Description: Records a TUI session and generates multi-format reports (HTML, Markdown, XML).
    *   Flags:
        *   `--out`: Output directory for recordings (default: timestamped directory).
