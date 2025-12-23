`tend` provides features for interactive debugging that allow you to pause test execution and interact with the sandboxed test environment. These features are built on tmux, enabling both manual and programmatic control over test sessions.

### 1. Debug Sessions (`tend run --debug-session`)

The primary mechanism for interactive debugging is the `--debug-session` flag. When running a single scenario with this flag, `tend` creates a dedicated, multi-window tmux session. This mode automatically enables interactive stepping and disables cleanup, preserving the test environment for inspection.

```bash
tend run my-tui-test --debug-session
```

This command creates a tmux session named `tend_<scenario-name>` with the following windows:

*   **`runner`**: Shows the test runner's step-by-step progress. Execution pauses at each step, waiting for user input. Press `Enter` to advance the test.
*   **`editor_test_dir`**: An instance of Neovim opened to the test's temporary root directory, allowing inspection of the sandboxed filesystem.
*   **`editor_test_steps`**: An instance of Neovim opened to the scenario's source file. As the test advances in the `runner` window, the editor automatically jumps to the source code of the current step.
*   **`term`**: An interactive shell with the exact sandboxed environment (PATH, HOME, XDG variables) that the test uses. This is the primary window for exploring the state of the system under test, running commands, or launching TUIs manually.
*   **`logs`**: A live view of `grove-core` logs.

The `editor` windows use your user environment to load your configuration, while the `term`, `runner`, and `logs` windows use the test's sandboxed environment.

By default, the session is created in the main tmux server. You can use the `--server=dedicated` flag to create it in a separate `tend-debug` tmux server instance.

### 2. Partial Execution (`--run-steps`)

To reach a specific point in a long scenario without manual stepping, you can use the `--run-steps` flag. This flag automatically executes the setup phase and a specified number of test steps, then pauses in interactive mode.

```bash
# Auto-run Setup plus test steps 1, 2, and 3, then pause at step 4
tend run my-long-test --run-steps=1,2,3
```

This is useful for quickly getting to a specific state in a test to begin debugging.

### 3. Exploring with `tend sessions`

For programmatic interaction with a running debug session, `tend` provides the `sessions` subcommand group. These commands are useful for scripting interactions or for LLM agents to explore a TUI.

*   **`tend sessions list`**: Lists all active `tend` debug sessions.
*   **`tend sessions capture <session-target>`**: Captures and prints the current text content of a tmux pane. It strips ANSI codes by default for easier parsing. Use `--wait-for <text>` to poll until specific text appears.
*   **`tend sessions send-keys <session-target> -- [keys...]`**: Sends keystrokes to a tmux pane to interact with a TUI.
*   **`tend sessions kill [session-name...]`**: Kills one or more `tend` debug sessions.

### 4. Recording with `tend record`

The `tend record` command captures a TUI interaction and generates reports. It launches a specified command within a recordable sub-shell, saving all keystrokes and terminal output.

```bash
tend record --out my-session -- my-app tui
```

This command creates multiple output files from the recording:

*   **`my-session.html`**: An interactive HTML file for replaying the session.
*   **`my-session.md`**: A Markdown report with plain text frames.
*   **`my-session.ansi.md`**: A Markdown report preserving ANSI color codes.
*   **`my-session.xml`**: An XML report structured for LLM consumption.
*   **`my-session.ansi.xml`**: An XML report with ANSI codes.

This feature is useful for creating documentation of a TUI workflow or for providing context to an LLM to assist in writing an automated test.