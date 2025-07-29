# Grove Tend Testing Framework

This directory contains a Go-based testing framework for Grove, replacing the previous ad-hoc bash scripts with a structured, maintainable solution.

## Project Structure

```
tend/
├── harness/           # Core test execution framework
│   ├── harness.go     # Core types and interfaces
│   └── errors.go      # Custom error types
├── pkg/               # Reusable helper packages
│   ├── fs/            # Filesystem utilities
│   │   ├── fs.go      # Basic file operations
│   │   ├── temp.go    # Temporary directory management
│   │   ├── grove.go   # Grove configuration helpers
│   │   ├── testdata.go# Test data generators
│   │   └── utils.go   # Additional utilities
│   └── git/           # Git operation helpers
│       ├── git.go     # Core git operations
│       ├── worktree.go# Worktree management
│       ├── remote.go  # Remote operations
│       ├── config.go  # Configuration helpers
│       ├── helpers.go # High-level test helpers
│       └── utils.go   # Utility functions
├── scenarios/         # Test scenario definitions
│   └── agent/         # Agent-specific tests (placeholder)
├── cmd/               # CLI applications
│   └── runner/        # Test runner CLI (placeholder)
├── go.mod             # Go module definition
└── example_test.go    # Example usage and verification tests
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

## Design Principles

- **Type Safety** - Leverage Go's type system to prevent runtime errors
- **Clear Error Messages** - Provide detailed context when things go wrong
- **Composability** - Build complex tests from simple, reusable components
- **Clean Architecture** - Separate concerns with clear interfaces
- **Testability** - All components can be unit tested
- **Resource Management** - Proper cleanup with defer statements