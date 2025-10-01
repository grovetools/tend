<!-- DOCGEN:OVERVIEW:START -->

<img src="docs/images/grove-tend-readme.svg" width="60%" />

Grove Tend is a Go library for creating scenario-based end-to-end testing frameworks. It is designed with a library-first philosophy, allowing developers to build a custom test runner binary tailored to their project's needs. This approach uses Go code for test definitions and logic, keeping them within the project's codebase.

<!-- placeholder for animated gif -->

## Key Features

-   **Scenario-Based Testing**: Organizes tests into `Scenario` structs composed of sequential `Step`s that share a `Context` object.
-   **Helper Packages**: Provides packages for filesystem interactions (`fs`), Git repository management (`git`), command execution (`command`), and assertions (`assert`).
-   **Mocking**: Supports defining mocks as Go binaries. The test harness can be configured to substitute these binaries for real dependencies during test execution.
-   **TUI Testing**: Automates `tmux` sessions to launch, send keystrokes to, and assert on the state of Terminal User Interface (TUI) applications.
-   **Interactive Debugging**: Includes interactive (`-i`) and debug (`-d`) modes. These modes pause execution before each step and allow developers to inspect state or manually interact with TUI sessions.

## Use Cases

-   **CLI Tool Testing**: Validating commands with file I/O, environment variables, and exit codes.
-   **Integration Testing**: Orchestrating and testing tools that depend on other CLI programs like `git`, `docker`, or `kubectl`.
-   **TUI Applications**: Automating tests for terminal UIs.
-   **Workflow Validation**: Verifying multi-step processes that require state to be maintained across actions.
-   **LLM-Generated Test Suites**: The framework's structure is intended for generation and maintenance by language models.

## How It Works

The core of Grove Tend is a test `Harness` that executes scenarios. The workflow is as follows:

1.  **Test Runner Creation**: A developer creates a main entry point for tests (e.g., `tests/e2e/main.go`) that imports the `grove-tend` library.
2.  **Scenario Definition**: Test cases are defined as `harness.Scenario` structs, each containing one or more `harness.Step`s.
3.  **Step Execution**: The harness runs each step sequentially. Each step function receives a `harness.Context` object, which manages a temporary directory for the test and provides a key-value store for passing state between steps.
4.  **Command Execution**: The `Context` provides a `Command()` factory. When mocks are enabled, this factory ensures that calls to external tools are routed to the specified mock binaries.
5.  **Cleanup**: After a scenario completes, the harness cleans up temporary resources, such as directories and `tmux` sessions, unless disabled for debugging.
6.  **CLI Interface**: The compiled test runner is a command-line application that can list and run scenarios, with filtering by name or tags.

## Role in the Grove Ecosystem

Grove Tend is used for end-to-end testing across command-line tools within the Grove ecosystem. This provides a consistent framework for writing and maintaining tests for any tool in the ecosystem, which is used for validating cross-tool interactions.

## LLM-Oriented Design

The structure of Grove Tend tests is intentionally verbose to be generated and maintained by Large Language Models (LLMs). This explicitness is designed to make the tests machine-readable for an LLM to comprehend, modify, and extend. This design choice facilitates the creation of E2E test suites that cover user workflows that may be impractical to write and maintain manually. The tests serve as machine-readable documentation of the system's behavior.

## Interactive Debugging

The framework includes features to assist in debugging E2E test failures.

-   The interactive (`-i`) flag pauses execution before each step, prompting the user to continue, skip, or quit. For TUI tests, it adds an option to attach to the live `tmux` session for manual interaction.
-   The debug (`-d`) flag is a shorthand that enables interactive mode, disables cleanup of temporary files, and splits the current `tmux` window, opening a new pane in the test's temporary directory. This allows developers to watch the test run on one side while inspecting files on the other.

---

### Installation

Grove-tend is a Go library. Add it to your project:
```bash
go get github.com/mattsolo1/grove-tend
```

For CLI usage, install via Grove meta-CLI:
```bash
grove install tend
```

Verify CLI installation:
```bash
tend version
```

See the [Grove Installation Guide](https://github.com/mattsolo1/grove-meta/blob/main/docs/02-installation.md) for setup.

<!-- DOCGEN:OVERVIEW:END -->

<!-- DOCGEN:TOC:START -->

See the [documentation](docs/) for detailed usage instructions:
- [Overview](docs/01-overview.md)
- [Examples](docs/02-examples.md)
- [Conventions](docs/03-conventions.md)

<!-- DOCGEN:TOC:END -->
