`grove-tend` is a scenario-based testing framework designed for end-to-end validation of command-line tools and Terminal User Interfaces (TUIs). It provides a structured environment for defining test cases in Go, managing isolated filesystems, and interacting with application subprocesses. Its core components are Scenarios, Steps, the Harness, and the Context.

Tests are defined as `Scenarios` composed of sequential `Steps`. The test `harness` executes these scenarios, managing state, sandboxed filesystems, mock dependencies, and automatic cleanup.

When executed, the `tend` CLI acts as a proxy. It discovers the project under test, builds a project-specific test binary containing that project's compiled `Scenario` definitions, and then executes that binary with the specified arguments.

Key capabilities include:

*   Hermetic test execution via temporary, sandboxed filesystems and home directories.
*   Mocking of command-line dependencies (e.g., `git`, `docker`, `kubectl`).
*   Programmatic control and state assertion for TUIs via managed tmux sessions.
*   Helpers for manipulating Git repositories, running commands, and managing Docker containers.
*   Interactive debugging modes for step-through execution and live TUI exploration.

### 1. Scenarios and Steps

The fundamental building blocks of a `grove-tend` test suite are `Scenarios` and `Steps`.

*   A **`Scenario`** is the top-level container for a single test case, representing a complete user workflow or feature validation. It groups a series of actions and assertions. Scenarios are defined using the `harness.NewScenario` constructor.
*   A **`Step`** is a single, named function within a `Scenario`. Each step performs a discrete action, such as setting up a file, running a command, or verifying output. It receives a `Context` object, which provides the API for interacting with the test environment.

```go
// A minimal scenario with a single step
var MyFirstScenario = harness.NewScenario(
    "my-first-scenario",
    "Verifies the basic functionality of a command.",
    []string{"smoke"}, // Tags for filtering
    []harness.Step{
        harness.NewStep("Run command and check output", func(ctx *harness.Context) error {
            // Test logic goes here
            result := ctx.Command("echo", "hello").Run()
            return result.AssertStdoutContains("hello")
        }),
    },
)
```

### 2. Scenario Configuration

The `harness.Scenario` struct provides several fields to control test execution and organization.

*   `Name` and `Description`: Strings used for identifying the test case in logs and reports.
*   `Tags`: A slice of strings for categorizing scenarios. Tests can be filtered using the `--tags` flag (e.g., `tend run --tags=smoke`).
*   `Setup`: A slice of `Step`s that run once before the main test steps. This phase is used for prerequisite tasks like setting up mocks or preparing the test filesystem.
*   `Steps`: The primary sequence of `Step`s that define the core logic of the test case.
*   `Teardown`: A slice of `Step`s that run after the main steps have completed, even if a failure occurred. This is used for cleanup tasks that must be performed regardless of the test outcome.
*   `LocalOnly`: A boolean that, when `true`, causes the scenario to be skipped in CI environments. This is for tests that depend on a local developer machine setup. The `--include-local` flag can override this behavior.
*   `ExplicitOnly`: A boolean that, when `true`, prevents the scenario from running as part of a general test run (e.g., `tend run`). It must be invoked explicitly by name (`tend run <scenario-name>`) or with the `--explicit` flag. This is for long-running, resource-intensive, or destructive tests.

### 3. The Harness

The `Harness` is the engine that executes scenarios. It is an internal component of the `tend` CLI that manages the entire test lifecycle.

Its primary responsibilities include:
*   **Isolation:** Creating a new, temporary root directory for each scenario run to ensure tests are hermetic.
*   **Context Management:** Instantiating the `Context` object passed to each step, populating it with paths to the isolated directories and other test state.
*   **Lifecycle Execution:** Executing the `Setup`, `Steps`, and `Teardown` phases in the correct order.
*   **Cleanup:** Automatically removing all temporary directories and resources created during a test run, unless disabled with the `--no-cleanup` flag for debugging.

### 4. The Context (`harness.Context`)

The `harness.Context` object is the primary API for test authors. It is passed to every `Step` function and provides methods for interacting with the sandboxed test environment.

Key functionalities include:

*   **Filesystem Management:** Create and retrieve paths to sandboxed directories.
    *   `NewDir`, `Dir`: Manage named subdirectories within the test's root.
    *   `HomeDir`, `ConfigDir`, `DataDir`, `CacheDir`: Access paths to a sandboxed user home directory structure (`$HOME`, `$XDG_CONFIG_HOME`, etc.).
*   **State Management:** Share data between steps within the same scenario.
    *   `Set(key, value)`: Store a value.
    *   `Get(key)`, `GetString(key)`, `GetInt(key)`, etc.: Retrieve stored values with type safety.
    *   `HasKey(key)`, `Keys()`: Introspect the stored state.
*   **Command Execution:** Run external commands within the sandboxed environment.
    *   `Command(name, args...)`: Creates a command that automatically runs with a `PATH` that prioritizes mock binaries set up for the test.
    *   `Bin(args...)`: A convenience wrapper for running the main binary of the project under test.
*   **Assertions:** Perform validations.
    *   `Check(description, err)`: A "fail-fast" or hard assertion. If the error is not `nil`, the step fails immediately.
    *   `Verify(func(v *verify.Collector))`: A "collecting" or soft assertion. It gathers multiple failures within its function block and reports them all at once without stopping on the first failure.
*   **TUI Control:** Launch and interact with TUI applications.
    *   `StartTUI(...)`: Starts a TUI application in an isolated `tmux` session, returning a `Session` handle for programmatic interaction (sending keys, capturing screen content).
    *   `StartHeadless(...)`: Runs a `bubbletea` model in a test mode without a real terminal, allowing for direct inspection of its state and view output.
