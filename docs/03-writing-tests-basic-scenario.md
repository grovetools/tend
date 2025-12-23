Of course. Here is a complete, practical example of a simple test scenario, adhering to the style guide.

***

### 1. Scenario Goal

This example demonstrates how to write a basic end-to-end test for a command-line tool. The tool, which we'll assume is called `greeter`, takes an `--output` flag and writes the message "Hello, World!" to the specified file. The goal is to verify that the tool runs successfully and that the output file contains the correct content.

### 2. The `tests/e2e/main.go` Entrypoint

Every `tend` test suite has a main entrypoint, typically located at `tests/e2e/main.go`. This file is responsible for collecting all the test scenarios and executing the test runner.

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
	// Collect all scenarios for this test suite into a slice.
	// As you add more scenario files, you'll add their constructors here.
	scenarios := []*harness.Scenario{
		GreeterScenario(),
	}

	// Create a cancellable context for graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Execute the test runner with the collected scenarios.
	// The `app` package handles CLI parsing, UI rendering, and scenario execution.
	if err := app.Execute(ctx, scenarios); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

### 3. The Scenario File

This file defines the actual test logic. It contains a `Scenario` with multiple `Steps`. Each step is a self-contained function that performs an action and/or an assertion.

```go
// File: tests/e2e/scenarios_greeter.go
package main

import (
	"path/filepath"

	"github.com/mattsolo1/grove-tend/pkg/fs"
	"github.com/mattsolo1/grove-tend/pkg/harness"
)

// GreeterScenario defines the test for our greeter CLI tool.
func GreeterScenario() *harness.Scenario {
	// Use harness.NewScenario to define the test's metadata.
	// This constructor automatically captures the file and line number,
	// which enables navigation features in debug mode.
	return harness.NewScenario(
		"greeter-writes-file",
		"Tests that the greeter CLI tool correctly writes a message to a file.",
		[]string{"smoke", "cli"}, // Tags for filtering tests.
		[]harness.Step{
			// Each test is composed of one or more steps.
			// Use harness.NewStep to define each step with a clear name.
			harness.NewStep("Run the greeter command", func(ctx *harness.Context) error {
				// Define the path for our output file within the test's sandboxed
				// temporary directory, accessible via ctx.RootDir.
				outputPath := filepath.Join(ctx.RootDir, "greeting.txt")

				// Store the path in the context to share it with the next step.
				ctx.Set("output_file", outputPath)

				// Use ctx.Bin() to run the project's main binary, which `tend`
				// discovers from the project's grove.yml file.
				// This creates a command: `greeter --output /tmp/tend-test-XXXX/greeting.txt`
				cmd := ctx.Bin("--output", outputPath)
				result := cmd.Run()

				// Use ctx.Check() for a "hard assertion". If this fails, the
				// step stops immediately. AssertSuccess checks for exit code 0.
				return ctx.Check("greeter command runs successfully", result.AssertSuccess())
			}),

			harness.NewStep("Verify the output file content", func(ctx *harness.Context) error {
				// Retrieve the output file path saved in the previous step.
				outputPath := ctx.GetString("output_file")

				// Use fs.AssertContains to check the file's content.
				// ctx.Check logs the assertion result and returns an error on failure.
				return ctx.Check(
					"output file contains 'Hello, World!'",
					fs.AssertContains(outputPath, "Hello, World!"),
				)
			}),
		},
	)
}
```