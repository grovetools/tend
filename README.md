# grove-tend

<img src="https://github.com/user-attachments/assets/c6ca0fe6-38b5-4f55-b0e8-7e42d5940d48" width="60%" />

[![CI](https://github.com/mattsolo1/grove-tend/actions/workflows/ci.yml/badge.svg)](https://github.com/mattsolo1/grove-tend/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/mattsolo1/grove-tend)](https://goreportcard.com/report/github.com/mattsolo1/grove-tend)
[![Go Reference](https://pkg.go.dev/badge/github.com/mattsolo1/grove-tend.svg)](https://pkg.go.dev/github.com/mattsolo1/grove-tend)

A Go library for creating powerful, scenario-based end-to-end testing frameworks.

**Grove Tend** provides the building blocks to replace ad-hoc bash scripts with structured, maintainable, and debuggable Go code. It is designed as a pure library, allowing you to build a custom testing CLI tailored to your project's needs.

---

## Features

-   **Library-First Design**: Import `grove-tend` to build your own test runner binary, keeping test definitions close to your code.
-   **Scenario-Based Testing**: Structure tests logically with `Scenarios`, `Steps`, and a shared `Context`.
-   **Rich Helper Packages**: Leverage built-in helpers for filesystem operations (`fs`), Git (`git`), command execution (`command`), Docker (`docker`), assertions (`assert`), and waiting (`wait`).
-   **Interactive Debugging**: Step through scenarios one-by-one with interactive mode (`-i`) or use the powerful debug mode (`-d`) for tmux integration.
-   **Beautiful Terminal UI**: Get clear, styled output with progress indicators, status updates, and command output boxes.
-   **Project-Specific Binary Discovery**: The globally installed `tend` binary will automatically find and execute a project-specific test binary, ensuring you always run the correct tests.
-   **CI-Friendly Reporting**: Generate JUnit, JSON, and GitHub Actions annotations for seamless CI/CD integration.
-   **First-Class Mocking Support**: Define mocks in Go, build them as binaries, and seamlessly swap between mocked and real dependencies.

## Getting Started: Using Tend as a Library

The primary way to use `tend` is to create a custom test binary within your own project.

### 1. Installation

Add `grove-tend` to your Go project:

```bash
go get github.com/mattsolo1/grove-tend
```

### 2. Create Your Test Runner

Create a new `main.go` file for your test runner (e.g., in `cmd/tester/main.go`):

```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/mattsolo1/grove-tend/pkg/app"
    "github.com/mattsolo1/grove-tend/pkg/command"
    "github.com/mattsolo1/grove-tend/pkg/fs"
    "github.com/mattsolo1/grove-tend/pkg/harness"
)

// Define a scenario specific to your project
var MyWebAppScenario = &harness.Scenario{
    Name:        "webapp-smoke-test",
    Description: "Performs a basic smoke test on the web application.",
    Tags:        []string{"smoke", "webapp"},
    Steps: []harness.Step{
        // Use a step builder to create a step
        harness.NewStep("Setup test directory", func(ctx *harness.Context) error {
            // The context manages a temporary directory for the scenario
            testDir := ctx.NewDir("webapp-test")
            ctx.Set("test_dir", testDir) // Store values for later steps
            return fs.WriteBasicGroveConfig(testDir)
        }),
        {
            Name: "Run a command",
            Func: func(ctx *harness.Context) error {
                cmd := command.New("echo", "Hello, Tend!").Dir(ctx.Dir("webapp-test"))
                result := cmd.Run()
                if result.Error != nil {
                    return result.Error
                }
                // Display formatted command output in verbose mode
                ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
                return nil
            },
        },
        // Use a built-in delay step
        harness.DelayStep("Wait for filesystem", 100*time.Millisecond),
    },
}

func main() {
    // Collect all scenarios for your test runner
    scenarios := []*harness.Scenario{
        MyWebAppScenario,
        // Add more scenarios here...
    }

    // Setup signal handling for graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <-sigChan
        fmt.Println("\nReceived interrupt, shutting down...")
        cancel()
    }()

    // Execute the tend application with your scenarios
    if err := app.Execute(ctx, scenarios); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

### 3. Build and Run Your Tests

Build your custom binary:

```bash
go build -o my-tests ./cmd/tester
```

Now you can use the CLI to run your scenarios:

```bash
# List all available scenarios
./my-tests list

# Run a specific scenario
./my-tests run webapp-smoke-test

# Run all scenarios tagged with 'smoke'
./my-tests run --tags=smoke

# Run interactively, stepping through each action
./my-tests run -i webapp-smoke-test

# Run in debug mode (implies interactive, no-cleanup, verbose, and tmux integration)
./my-tests run -d webapp-smoke-test
```

## The `tend` CLI

Your custom test binary is a full-featured CLI application with the following commands:

-   `run [scenario...]`: Executes test scenarios. Can be filtered by name (with glob patterns) or tags.
-   `list`: Lists all available scenarios in a table format, showing their names, descriptions, tags, and step counts.
-   `validate`: Parses and validates all scenario definitions to catch errors early.
-   `version`: Prints the version information of the test binary.

## Core Concepts

-   **`harness.Scenario`**: A collection of steps that defines a complete end-to-end test. It includes a name, description, and tags for organization and filtering.
-   **`harness.Step`**: A single action within a scenario. It consists of a name and a function that receives a `Context`.
-   **`harness.Context`**: A state container passed between steps in a scenario. It manages the temporary test directory and provides a key-value store for sharing data (e.g., file paths, command output) between steps.

## Mocking External Dependencies

`tend` provides a first-class solution for mocking external dependencies like `git`, `docker`, or other CLIs, moving beyond brittle shell scripts.

### Concept

Instead of writing mock logic in shell scripts inside your tests, you can write them in Go, compile them into mock binaries, and have `tend` manage them during test execution. This makes your mocks more robust, maintainable, and easier to debug.

### 1. Defining Mocks in Go

Create a directory for your mocks, for example, `tests/mocks/`. For each command you want to mock, create a subdirectory with a `main.go` file.

**Example: `tests/mocks/llm/main.go`**
```go
package main

import (
	"fmt"
	"os"
)

// A simple mock for an 'llm' command.
func main() {
	// Mocks can be stateful by reading/writing to files.
	// For this example, we just print a static response.
	fmt.Println("This is a mocked LLM response.")
	os.Exit(0)
}
```

### 2. Building Mocks

Add a target to your `Makefile` to automatically build your mock binaries and place them in your project's `./bin` directory, following the `mock-<name>` convention.

**Example `Makefile` target:**
```makefile
MOCK_SRC_DIR=tests/mocks
MOCK_BIN_DIR=bin
MOCKS=$(shell find $(MOCK_SRC_DIR) -mindepth 1 -maxdepth 1 -type d -exec basename {} \;)

build-mocks:
    @echo "Building mocks: $(MOCKS)"
    @mkdir -p $(MOCK_BIN_DIR)
    @for mock in $(MOCKS); do \
        echo "  -> Building mock $$mock"; \
        go build -o $(MOCK_BIN_DIR)/mock-$$mock $(MOCK_SRC_DIR)/$$mock; \
    done

# Add it to your main build target
build: build-mocks
    # ... your main binary build ...
```

### 3. Using Mocks in Scenarios

Use the `harness.SetupMocks` step builder to activate your mocks for a scenario. `tend` will create a temporary `bin` directory for the test run, symlink your mocks into it, and add it to the `PATH`.

```go
var MyScenario = &harness.Scenario{
    Name: "my-feature-test",
    Steps: []harness.Step{
        // Setup mocks for 'git' and 'llm'.
        // By convention, tend looks for ./bin/mock-git and ./bin/mock-llm.
        harness.SetupMocks(
            harness.Mock{CommandName: "git"},
            harness.Mock{CommandName: "llm"},
        ),
        // You can also create simple mocks with inline scripts.
        harness.SetupMocks(
            harness.Mock{
                CommandName: "kubectl",
                Script:      "#!/bin/bash\necho '{\"status\":\"mocked\"}'",
            },
        ),
        harness.NewStep("Run feature command", func(ctx *harness.Context) error {
            // Use ctx.Command() to create commands that respect the mock PATH.
            cmd := ctx.Command("./my-app-binary", "feature", "apply")
            result := cmd.Run()
            // ... assert on mock output ...
            return result.Error
        }),
    },
}
```

### 4. Swapping Mocks for Real Binaries (Integration Testing)

For integration tests, you can swap mocks with their real counterparts from your Grove ecosystem using the `--use-real-deps` flag. `tend` uses `grove dev current <tool>` to find the correct path to the active binary.

```bash
# Run with all mocks enabled (default)
./my-tests run my-feature-test

# Swap the 'git' mock for the real binary
./my-tests run my-feature-test --use-real-deps=git

# Swap 'git' and 'llm'
./my-tests run my-feature-test --use-real-deps=git,llm

# Swap all available mocks for their real counterparts
./my-tests run my-feature-test --use-real-deps=all
```

## Development

To work on `grove-tend` itself:

```bash
# Build the main binary
make build

# Run all linters and tests
make check

# Clean build artifacts
make clean
```

Tests for the framework can be found in the `tests/` directory.

---

Inspired by modern testing frameworks and built for the Grove ecosystem.
