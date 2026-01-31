`tend` is a command-line tool and Go library for orchestrating end-to-end tests for CLI applications and Terminal User Interfaces (TUIs). It executes scenarios defined in Go code within hermetic, sandboxed environments, managing lifecycle events, process isolation, and dependency mocking.

## Core Mechanisms

**Proxy Execution**: When `tend` is executed in a project directory, it detects the project-specific test runner (typically in `tests/e2e/tend`), compiles a binary containing the project's scenarios, and delegates execution to that binary. This allows tests to be versioned with the source code while maintaining a unified CLI experience.

**Environment Sandboxing**: Each scenario runs in a temporary directory. The harness automatically overrides environment variables (`HOME`, `XDG_CONFIG_HOME`, `XDG_DATA_HOME`) to point to isolated paths within this temporary workspace. This prevents tests from reading or modifying the host user's configuration files.

**TUI Integration**: `tend` utilizes `tmux` to run interactive terminal applications in the background. It provides an API to send keystrokes to a session and inspect the screen content (text and ANSI codes) for assertions. This enables black-box testing of complex TUI interactions without requiring a visible terminal window.

**Mocking**: The harness creates a temporary `bin` directory on the `PATH`. Tests can register mock implementations for external dependencies (e.g., `git`, `docker`, `kubectl`). These mocks trap calls and return controlled output, isolating the test from system tools and network resources.

## Features

### Test Execution
*   **`tend run`**: Executes specific scenarios. Supports filtering by tags or name.
*   **`tend tui`**: Launches an interactive terminal interface for browsing, filtering, and executing available test scenarios.
*   **`tend ecosystem run`**: Discovers and executes E2E test suites across multiple projects in a workspace, running them in parallel and aggregating results.

### Debugging & Observability
*   **Interactive Debugging**: The `--debug` flag launches the test in a visible `tmux` session, pausing execution to allow manual inspection of the filesystem and process state.
*   **Session Management**: `tend sessions` lists and manages background `tmux` sessions created by test runs or debug sessions.
*   **Recording**: `tend record` captures terminal output and keystrokes from a command execution, saving the session as HTML, Markdown, or JSON for documentation or analysis.

### Demo Environments
*   **`tend demo create`**: Generates isolated, self-contained Grove ecosystems (e.g., "homelab") populated with synthetic repositories, notes, and plans. These environments are used for generating consistent screenshots and demonstrating tool capabilities without exposing private data.

## Integration

`tend` is designed to test the Grove ecosystem itself but can be used for any CLI tool.

*   **`flow`**: Tests validate the orchestration of agents and job execution by inspecting file modifications and agent log outputs.
*   **`nav`**: Tests verify `tmux` session management by asserting against the state of the backend `tmux` server.
*   **`nb`**: Tests confirm note creation and retrieval by inspecting the sandboxed filesystem.

