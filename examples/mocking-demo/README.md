# Grove Tend Mocking Demo

This example demonstrates the powerful mocking capabilities of Grove Tend, showing how to:

1. Define mocks as Go binaries instead of shell scripts
2. Use inline script mocks for simple cases
3. Seamlessly swap between mocked and real dependencies
4. Test complex workflows with external tool dependencies

## Structure

```
mocking-demo/
├── main.go                 # Test scenarios using mocks
├── Makefile               # Build automation
├── tests/mocks/           # Mock implementations
│   ├── git/main.go       # Git command mock
│   ├── docker/main.go    # Docker command mock
│   └── llm/main.go       # LLM CLI mock
└── bin/                   # Built binaries (after build)
```

## Quick Start

1. **Build everything** (mocks and test binary):
   ```bash
   make build
   ```

2. **List available scenarios**:
   ```bash
   make list
   ```

3. **Run all scenarios**:
   ```bash
   make test
   ```

## Available Scenarios

### 1. Git Workflow (`git-workflow`)
Tests a typical git workflow using a mocked git command:
- Initialize repository
- Check status
- Stage and commit files

```bash
make run-git
```

### 2. Docker Operations (`docker-operations`)
Tests docker commands using a mocked docker CLI:
- Check docker version
- List images
- Pull an image
- List running containers

```bash
make run-docker
```

### 3. LLM Integration (`llm-integration`)
Demonstrates both binary mocks and inline script mocks:
- Query an LLM with various prompts
- Test JSON output mode
- Use a simple inline script mock

```bash
make run-llm
```

### 4. Mixed Dependencies (`mixed-dependencies`)
Shows how multiple tools work together in an integration test:
- Creates a deployment script using git, docker, and kubectl
- Demonstrates how to selectively use real binaries

```bash
make run-mixed
```

## Using Real Dependencies

The `--use-real-deps` flag allows you to swap mocks for real binaries from your Grove ecosystem:

### Run with real git binary only:
```bash
make run ARGS='git-workflow --use-real-deps=git'
```

### Run with multiple real binaries:
```bash
make run ARGS='mixed-dependencies --use-real-deps=git,docker'
```

### Run with all real binaries:
```bash
make run-real-all
```

## Mock Implementations

### Binary Mocks (Go)

Each mock is a standalone Go program that simulates the behavior of the real tool:

- **git mock**: Simulates repository operations with realistic output
- **docker mock**: Provides container and image management responses
- **llm mock**: Returns context-aware responses based on prompts

### Inline Script Mocks

For simple cases, you can define mocks directly in your test:

```go
harness.SetupMocks(
    harness.Mock{
        CommandName: "simple-tool",
        Script: `#!/bin/bash
echo "Output: $@"`,
    },
)
```

## Interactive and Debug Modes

### Interactive Mode
Step through scenarios one at a time:
```bash
make interactive ARGS='git-workflow'
```

### Debug Mode
Full debugging with tmux integration:
```bash
make debug ARGS='docker-operations'
```

## How It Works

1. **Mock Discovery**: Tend looks for mocks in `./bin/mock-<command>`
2. **PATH Manipulation**: A temporary bin directory is added to PATH for each test
3. **Command Execution**: Use `ctx.Command()` to ensure mocks are found
4. **Real Binary Discovery**: Uses `grove dev current <tool>` to find real binaries

## Tips

- Mock binaries can be stateful by reading/writing files
- Use stderr for debug logging (won't interfere with output assertions)
- The `ctx.Command()` helper ensures proper PATH setup
- Combine `--use-real-deps` with verbose mode (`-v`) to see which binaries are used

## Extending This Example

To add a new mock:

1. Create a new directory under `tests/mocks/`
2. Add a `main.go` file implementing the mock behavior
3. Run `make build-mocks` to compile it
4. Use it in a scenario with `harness.Mock{CommandName: "your-tool"}`

This example showcases how Grove Tend makes it easy to write maintainable, robust tests for tools that depend on external commands.