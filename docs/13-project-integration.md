The `tend` command-line tool uses a proxy architecture to build and execute project-specific test binaries. This design keeps test suites self-contained and allows them to be compiled with direct access to a project's internal packages.

### 1. The Proxy Binary Concept

The `tend` binary installed in your `PATH` is a lightweight proxy. Its primary function is to detect the project context, compile a dedicated test runner for that project, and then transfer execution to it.

The execution flow is as follows:

1.  When a command like `tend run` is executed, the proxy binary first checks if the command can be run without project context (e.g., `tend version`, `tend sessions`). If not, it proceeds with the proxy logic.
2.  It searches upward from the current directory for a `grove.yml` file to identify the project root.
3.  Once the project root is found, it looks for test runner source code in a conventional location, typically `tests/e2e/`.
4.  It invokes the Go compiler (`go build`) to compile the test sources into a project-specific executable. This binary is usually placed in `bin/tend-e2e`. During this build, it also compiles any mock binaries defined in `tests/e2e/mocks/src/`.
5.  Using a `syscall.Exec` call, the proxy binary replaces its own process with the newly compiled test runner, passing along the original command-line arguments. An environment variable, `TEND_IS_CHILD_PROCESS`, is set to prevent the new process from re-triggering the proxy logic.
6.  The project-specific test runner, now executing, has all the project's scenarios compiled into it and proceeds to handle the original command (e.g., `run`, `list`).

### 2. Architectural Benefits

This proxy architecture provides several mechanical advantages:

*   **Self-Contained Tests**: Scenarios are not discovered from source files at runtime. Instead, they are compiled directly into the project's test binary. This eliminates the need for a central test registry and ensures test suites are entirely self-contained within their project.
*   **Direct Package Access**: Because the test runner is compiled as part of the project, its scenarios can directly import and use the project's internal packages, structs, and helper functions without violating Go's package visibility rules.
*   **Automatic Scenario Discovery**: Commands like `tend list` function by reading a slice of `*harness.Scenario` pointers that were collected and compiled into the binary. The discovery happens at compile time, not runtime.
*   **Dependency Isolation**: Each project's test runner is a distinct Go application. Its dependencies are managed by a `go.mod` file within the test source directory (e.g., `tests/e2e/go.mod`), isolating them from other projects.

### 3. The Test Entrypoint Structure

A project's test runner is built from a conventional `main.go` file located in the test source directory (e.g., `tests/e2e/main.go`). This file serves as the entrypoint for the compiled binary.

Its responsibilities are to:
1.  Import the Go packages that define the test scenarios.
2.  Collect all scenario definitions (`*harness.Scenario`) into a single slice.
3.  Pass this slice to `app.Execute()`, which transfers control to the `tend` framework.

The `tend` framework is responsible for providing the CLI, including the `run`, `list`, and `tui` commands. The project's entrypoint does not need to implement any command-line logic itself.

A minimal entrypoint has the following structure:

```go
// File: tests/e2e/main.go
package main

import (
	"context"
	"os"

	"github.com/mattsolo1/grove-tend/pkg/app"
	"github.com/mattsolo1/grove-tend/pkg/harness"
)

func main() {
	// Collect all scenarios defined in the project's test files.
	scenarios := []*harness.Scenario{
		GitWorkflowScenario,
		DockerScenario,
		// ... add other scenarios here
	}

	// Hand control to the tend application framework.
	if err := app.Execute(context.Background(), scenarios); err != nil {
		os.Exit(1)
	}
}
```

### 4. Build Process Details

The proxy binary automates the build process using standard Go tooling.

*   **Compilation**: It executes `go build` targeting the test source directory (e.g., `tests/e2e/`). Build flags (`-ldflags`) are used to inject version information into the binary for tracking and debugging.
*   **Output**: The resulting executable is placed in a predictable location, typically `bin/tend-e2e`. The name is altered to avoid conflicts if the project being tested is `tend` itself.
*   **Execution**: The build is triggered automatically on every relevant `tend` command invocation, ensuring that the test runner is always up-to-date with the latest source code changes.
*   **Error Handling**: If the `go build` command fails, its output is captured and printed to standard error, halting the process immediately with a clear message.

### 5. Integration Points

Projects integrate with this system through simple `Makefile` targets. The `tend` proxy encapsulates all the complex discovery and build logic, allowing the project's `Makefile` to remain minimal. The target simply invokes the `tend` command, relying on the proxy to handle the rest.

A typical integration looks like this:

```makefile
# Makefile
test-e2e:
	@tend run $(ARGS)
```