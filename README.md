# Grove Tend Testing Framework

Grove Tend is a Go-based testing framework library for the Grove ecosystem. It provides a structured, maintainable solution for writing end-to-end tests, replacing ad-hoc bash scripts with proper error handling, cleanup, and beautiful output.

**Note:** Grove Tend is now a pure library. Test scenarios should be defined in the repositories they test (e.g., grove-sandbox defines its own agent tests).

## Project Structure

```
grove-tend/
├── internal/
│   ├── harness/       # Core test execution framework
│   │   ├── harness.go # Core types and interfaces
│   │   └── ...        # Other harness components
│   └── cmd/           # CLI implementation
│       ├── root.go    # Root command setup
│       ├── run.go     # Run command
│       ├── list.go    # List command
│       └── validate.go# Validate command
├── pkg/               # Public API - reusable helper packages
│   ├── app/           # Application builder
│   │   └── app.go     # Main entry point for custom binaries
│   ├── fs/            # Filesystem utilities
│   ├── git/           # Git operation helpers
│   ├── command/       # Command execution helpers
│   ├── ui/            # Terminal UI components
│   └── ...            # Other helper packages
├── examples/          # Example usage
│   └── custom-tend/   # Example of using tend as a library
│       └── main.go    # Complete example with all scenarios
├── main.go            # Main binary (empty - for library demonstration)
├── go.mod             # Go module definition
└── Makefile           # Build automation
```

## Implementation Status

### ✅ Completed (Sessions 01-05)

1. **Core Abstractions** - Defined the foundational types and interfaces:
   - `Context` - Carries state through scenario execution with thread-safe operations
   - `Step` - Represents a single test action
   - `Scenario` - Collection of steps defining a test
   - `Harness` - Test execution engine with interactive and batch modes
   - `Result` - Test outcome representation with detailed step results

2. **Filesystem Helpers** - Robust filesystem utilities:
   - Safe file/directory operations
   - Temporary directory management with cleanup
   - Grove configuration file generation
   - Test data and project structure creation

3. **Git Helpers** - Type-safe Git operations:
   - Repository initialization and basic operations
   - Worktree management for multi-branch testing
   - Remote repository operations
   - Test repository setup with sensible defaults

4. **Command Runner** - Execute shell commands with proper output capture:
   - Command execution with timeout handling
   - Streaming output support for long-running processes
   - Grove-specific command helpers
   - Docker container management utilities

5. **Harness Implementation** - Complete test execution logic:
   - Full scenario execution with step-by-step progress
   - Interactive mode with user prompts
   - Batch execution of multiple scenarios
   - Context state management between steps
   - Step builder utilities (retry, conditional, sequential)
   - Comprehensive error handling and cleanup

### 🔄 Next Steps (Sessions 06-10)

6. **UI Components** - Beautiful terminal interface with lipgloss
7. **Runner CLI** - Command-line interface with Cobra
8. **First Scenario** - Convert agent isolation test from bash
9. **CI Integration** - Reporting and GitHub Actions integration
10. **Advanced Helpers** - Wait utilities and verification helpers

## Usage Example

```go
// Define a test scenario
scenario := &harness.Scenario{
    Name:        "example-test",
    Description: "An example test scenario",
    Tags:        []string{"example", "demo"},
    Steps: []harness.Step{
        {
            Name: "Setup test environment",
            Func: func(ctx *harness.Context) error {
                testDir := ctx.NewDir("test")
                return fs.WriteBasicGroveConfig(testDir)
            },
        },
    },
}

// Execute with harness
h := harness.New(harness.Options{Verbose: true})
result, err := h.Run(context.Background(), scenario)
```

## Testing

Run the verification tests:

```bash
cd tend
go test -v
```

This will test the core abstractions, filesystem helpers, and git helpers to ensure everything is working correctly.

## Using Tend as a Library

Grove Tend can be used as a library in other repositories to define custom test scenarios. This allows you to leverage the testing framework while keeping your test definitions close to your code.

### Installation

Add grove-tend to your Go module:

```bash
go get github.com/mattsolo1/grove-tend
```

### Creating a Custom Test Binary

Create a new `main.go` file in your repository (e.g., `cmd/test/main.go`):

```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "github.com/mattsolo1/grove-tend/internal/harness"
    "github.com/mattsolo1/grove-tend/pkg/app"
)

// Define your custom scenarios
var MyScenario = &harness.Scenario{
    Name:        "my-test-scenario",
    Description: "Tests specific to my repository",
    Tags:        []string{"integration"},
    Steps: []harness.Step{
        harness.NewStep("Setup environment", func(ctx *harness.Context) error {
            // Your test logic here
            return nil
        }),
        // Add more steps...
    },
}

func main() {
    // List all your scenarios
    scenarios := []*harness.Scenario{
        MyScenario,
        // Add more scenarios...
    }

    // Setup signal handling
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <-sigChan
        fmt.Println("\nReceived interrupt signal, shutting down...")
        cancel()
    }()

    // Execute the tend application with your scenarios
    if err := app.Execute(ctx, scenarios); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

### Building and Running

Build your custom tend binary:

```bash
go build -o mytend ./cmd/test
```

Run your tests:

```bash
./mytend list                    # List all custom scenarios
./mytend run my-test-scenario    # Run specific scenario
./mytend run --tags=integration  # Run by tags
```

### Example

See the `examples/custom-tend` directory for a complete example of using tend as a library.

## Design Principles

- **Type Safety** - Leverage Go's type system to prevent runtime errors
- **Clear Error Messages** - Provide detailed context when things go wrong
- **Composability** - Build complex tests from simple, reusable components
- **Clean Architecture** - Separate concerns with clear interfaces
- **Testability** - All components can be unit tested
- **Resource Management** - Proper cleanup with defer statements