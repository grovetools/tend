# Instructions for: Core Concepts

Explain the fundamental components of `grove-tend`. Adhere to the style guide.

**Content Outline:**

1.  **Scenarios and Steps:**
    *   Explain that a `Scenario` is the top-level container for a test case, defined with `harness.NewScenario`.
    *   Explain that a `Step` is a single, named function within a scenario that receives a `Context`.
    *   Show a minimal code snippet of a `Scenario` with one `Step`.

2.  **Scenario Configuration:**
    *   Describe the `Scenario` struct fields:
        *   `Name` and `Description`: Identification and documentation.
        *   `Tags`: String slice for filtering scenarios (e.g., `--tags=smoke`).
        *   `Setup`: Steps that run before the main test steps (e.g., mock setup, file preparation).
        *   `Steps`: The main test logic.
        *   `Teardown`: Cleanup steps that run after the main steps (even on failure).
        *   `LocalOnly`: Skips the scenario in CI environments unless `--include-local` is passed.
        *   `ExplicitOnly`: Skips the scenario when running all tests unless explicitly named or `--explicit` is passed.

3.  **The Harness:**
    *   Describe the `Harness` as the test runner.
    *   Explain its responsibilities:
        *   Creating a temporary root directory for each scenario run.
        *   Instantiating the `Context` object for each step.
        *   Executing `Setup`, `Steps`, and `Teardown` phases in order.
        *   Performing automatic cleanup of temporary resources (unless disabled).

4.  **The Context (`harness.Context`):**
    *   Describe the `Context` as the primary API for interacting with the test environment from within a `Step`.
    *   List its key areas of functionality:
        *   **Filesystem Management:** Creating and accessing sandboxed directories (`NewDir`, `Dir`, `HomeDir`, `ConfigDir`, `DataDir`, `CacheDir`).
        *   **State Management:** Passing data between steps with `Set` and type-safe getters (`Get`, `GetString`, `GetStringSlice`, `GetInt`, `GetBool`). Also mention `HasKey` and `Keys` for state introspection.
        *   **Command Execution:** Running sandboxed commands with automatic mock resolution (`Command`, `Bin`).
        *   **Assertions:** Performing fail-fast (`Check`) or collecting (`Verify`) assertions.
        *   **TUI Control:** Launching and interacting with TUI applications (`StartTUI`, `StartHeadless`).
