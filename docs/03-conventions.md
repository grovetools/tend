# Conventions 

This document outlines the standard conventions for setting up, running, and maintaining end-to-end tests using the `grove-tend` framework.

### 1. The Test Runner Binary

Each project creates its own test runner binary by importing `grove-tend` as a library. This allows test scenarios to be defined and compiled directly within the project they are intended to test.

-   **Location**: The source code for the test runner is located in `tests/e2e/main.go`.
-   **Purpose**: The `main.go` file imports the `grove-tend` library, defines all project-specific `harness.Scenario`s, and registers them with the application framework.
-   **Execution**: The global `tend` CLI (installed via `grove install tend`) searches upwards from the current directory for a `grove.yml` file to identify the project root. It then looks for a project-specific test runner binary (conventionally at `./bin/tend`) and delegates execution to it. This ensures the correct tests are run for the current project context.

A typical test runner entrypoint contains:

```go
// File: tests/e2e/main.go
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
        ExampleBasicScenario(),
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

A standard set of `make` targets provides a consistent interface for building and running tests across projects.

-   **`build-mocks`**: This target compiles any Go-based mock binaries required for testing. It searches for mock source code in `tests/mocks/` and builds the executables into the project's `./bin` directory.

-   **`test-e2e-build`** (or `test-tend-build`): This target compiles the project's custom test runner binary from `tests/e2e/main.go` and places the executable in the `./bin` directory.

-   **`test-e2e`**: This is the primary command for running end-to-end tests. It depends on targets that build the main project, the mocks, and the test runner binary before execution. It then executes the compiled test runner.

-   **`ARGS` Variable**: The `ARGS` variable is used to pass arguments to the test runner from the command line.

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

Projects using `grove-tend` follow a conventional directory layout:

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

-   **`tests/e2e/`**: Contains end-to-end test code, including the `main.go` entrypoint and scenario definitions, which are often split into multiple `scenarios_*.go` files.
-   **`tests/mocks/`**: Contains the source code for custom mock binaries used to simulate external dependencies like `git`, `docker`, or other CLIs.
-   **`bin/`**: The output directory for compiled binaries, including the project's test runner and any mocks. This directory is typically added to `.gitignore`.

### 4. LLM-Friendly Patterns

The design of `grove-tend` tests is intentionally verbose to be machine-readable and maintainable by Large Language Models (LLMs).

-   **Descriptive Naming**: Scenario and Step names are strings that should clearly describe their purpose. This allows both developers and LLMs to understand the test flow.
-   **Clear Descriptions**: The `Description` field for both Scenarios and Steps is used to provide context on the "why" behind a test, complementing the "what" described by the name.
-   **Executable Documentation**: The step-by-step nature of `tend` scenarios serves as executable documentation of the system's expected behavior.

### 5. Development Workflow

Integrating `grove-tend` into the development cycle provides a structured approach to testing and debugging.

1.  **Write/Generate Tests**: Use an LLM to assist in generating new `harness.Scenario` definitions based on feature requirements or bug reports.
2.  **Run Tests**: Execute the test suite locally using `make test-e2e`.
3.  **Debug Failures**: If a test fails, use flags to diagnose the issue:
    -   `make test-e2e ARGS="run -i <scenario-name>"` to step through the failing scenario one action at a time.
    -   `make test-e2e ARGS="run -d <scenario-name>"` for a more complete debugging environment with `tmux` integration.
4.  **Iterate**: Fix the implementation code and re-run tests until they pass.
5.  **CI/CD Integration**: The `make test-e2e` command can be directly integrated into CI/CD pipelines. The framework automatically detects CI environments to produce machine-readable output formats like JUnit XML for reporting.
