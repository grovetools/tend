# Conventions for Integrating Grove Tend

This document outlines the standard conventions for setting up, running, and maintaining end-to-end tests using the `grove-tend` framework within the Grove ecosystem. Adhering to these patterns ensures consistency and leverages the framework's full capabilities.

### 1. The Test Runner Binary

Each project that uses `grove-tend` creates its own dedicated test runner binary. This approach treats `grove-tend` as a library, allowing test scenarios to be defined and compiled directly within the project they are intended to test.

-   **Location**: The source code for the test runner is typically located in `tests/e2e/main.go`.
-   **Purpose**: The primary role of this `main.go` file is to import the `grove-tend` library, define all project-specific `harness.Scenario`s, and register them with the `tend` application framework.
-   **Execution**: The global `tend` CLI (installed via `grove install tend`) is designed to be workspace-aware. When executed, it automatically searches for a project-specific test runner binary (conventionally at `./bin/tend` or a similar path). If found, it delegates execution to that binary, ensuring the correct tests are always run for the current project context.

A typical test runner entrypoint looks like this:

```go
// File: grove-context/tests/e2e/main.go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/mattsolo1/grove-tend/pkg/app"
    "github.com/mattsolo1/grove-tend/pkg/harness"
)

func main() {
    // A list of all E2E scenarios for the project.
    scenarios := []*harness.Scenario{
        BasicContextGenerationScenario(),
        MissingRulesScenario(),
        // ... more scenarios
    }

    // Execute the tend application with the project's scenarios.
    if err := app.Execute(context.Background(), scenarios); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

### 2. Makefile Integration

A standard set of `make` targets provides a consistent interface for building and running tests across all ecosystem projects.

-   **`build-mocks`**: This target compiles any Go-based mock binaries required for testing. It searches for mock source code, typically located in `tests/mocks/`, and builds the executables into the project's `./bin` directory.

-   **`test-e2e-build`** (or `test-tend-build`): This target compiles the project's custom test runner binary from `tests/e2e/main.go` and places the executable in the `./bin` directory.

-   **`test-e2e`**: This is the primary command for running end-to-end tests. It ensures all necessary components are built before execution. The typical dependency chain is: `test-e2e: build build-mocks test-e2e-build`. After building, it executes the compiled test runner.

-   **`ARGS` Variable**: To pass arguments to the test runner (such as scenario names or flags), the `ARGS` variable is used. This allows for flexible control from the command line.

**Example `Makefile` Usage:**
```bash
# Run all end-to-end tests
make test-e2e

# Run a specific scenario by name
make test-e2e ARGS="run my-scenario-name"

# Run all scenarios with the 'smoke' tag
make test-e2e ARGS="run --tags=smoke"

# Run a scenario in interactive mode for debugging
make test-e2e ARGS="run -i my-failing-scenario"
```

### 3. Project Structure

Projects using `grove-tend` follow a conventional directory layout to keep test-related files organized and discoverable.

```
<project-root>/
├── bin/
│   ├── tend          # Compiled project-specific test runner
│   └── mock-git      # Compiled mock binary for git
│   └── mock-docker   # Compiled mock binary for docker
├── tests/
│   ├── e2e/
│   │   ├── main.go      # Test runner entrypoint
│   │   └── scenarios.go # Scenario definitions
│   └── mocks/
│       ├── git/
│       │   └── main.go  # Source for the git mock binary
│       └── docker/
│           └── main.go  # Source for the docker mock binary
└── Makefile             # Defines build and test targets
```

-   **`tests/e2e/`**: Contains all end-to-end test code, including the `main.go` entrypoint and scenario definitions, which are often split into multiple `scenarios_*.go` files for organization.
-   **`tests/mocks/`**: Contains the source code for any custom mock binaries used to simulate external dependencies like `git`, `docker`, or other CLIs.
-   **`bin/`**: The standard output directory for compiled binaries, including the project's test runner and any mocks. This directory is typically added to `.gitignore`.

### 4. LLM-Friendly Patterns

The design of `grove-tend` tests is intentionally verbose to be easily understood, generated, and maintained by Large Language Models (LLMs).

-   **Descriptive Naming**: Scenario and Step names should be clear, human-readable phrases that describe their purpose. This helps both developers and LLMs understand the test flow at a glance.
-   **Clear Descriptions**: The `Description` field for both Scenarios and Steps should be used to provide context on the "why" behind a test, complementing the "what" described by the name.
-   **Living Documentation**: Well-written tests serve as living, executable documentation of the system's behavior. The detailed, step-by-step nature of `tend` scenarios makes them an excellent reference for how a feature is intended to work.

### 5. Development Workflow

Integrating `grove-tend` into the development cycle provides a structured approach to testing and debugging.

1.  **Write/Generate Tests**: Use an LLM to assist in generating new `harness.Scenario` definitions based on feature requirements or bug reports.
2.  **Run Tests**: Execute the test suite locally using `make test-e2e`.
3.  **Debug Failures**: If a test fails, use the interactive or debug flags to diagnose the issue:
    -   `make test-e2e ARGS="run -i <scenario-name>"` to step through the failing scenario one action at a time.
    -   `make test-e2e ARGS="run -d <scenario-name>"` for a more immersive debugging experience with `tmux` integration.
4.  **Iterate**: Fix the implementation code and re-run tests until they pass.
5.  **CI/CD Integration**: The `make test-e2e` command can be directly integrated into CI/CD pipelines. The framework automatically detects CI environments to produce machine-readable output formats like JUnit XML for reporting.