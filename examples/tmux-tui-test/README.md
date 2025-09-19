# TUI Testing with Tmux Example

This example demonstrates how to use grove-tend's TUI testing capabilities to test terminal user interfaces.

## Features

The example includes three test scenarios:

### 1. `example-tui-tmux` - Basic TUI Testing
- Creates a simple interactive TUI script
- Launches it in a tmux session
- Sends keystrokes and verifies responses
- Captures and validates TUI output

### 2. `example-bubbletea-headless` - Headless BubbleTea Testing
- Tests a BubbleTea model without tmux
- Fast unit testing for TUI logic
- Sends messages and verifies state changes
- No terminal required

### 3. `example-tui-interactive-debug` - Interactive Debugging
- Demonstrates the attach feature for manual debugging
- In interactive mode, testers can:
  - View current TUI state
  - Attach to the tmux session with 'a'
  - Manually interact with the TUI
  - Detach with 'Ctrl-b d' to continue testing

## Running the Tests

### Build the example:
```bash
go build -o tmux-tui-test
```

### Run all TUI tests:
```bash
./tmux-tui-test run all
```

### Run a specific test:
```bash
./tmux-tui-test run example-tui-tmux
```

### Run in interactive mode (enables attach):
```bash
./tmux-tui-test run example-tui-interactive-debug --interactive
```

When prompted during interactive mode:
- Press Enter to continue to the next step
- Press 'a' to attach to the tmux session
- Press 'q' to quit the test

### Run in verbose mode to see TUI captures:
```bash
./tmux-tui-test run example-tui-tmux --verbose
```

## Requirements

- `tmux` must be installed for the tmux-based tests
- The headless BubbleTea test doesn't require tmux

## Use Cases

This testing approach is ideal for:
- E2E testing of CLI tools with interactive modes
- Verifying TUI applications behave correctly
- Testing BubbleTea/Charm applications
- Automated testing in CI/CD pipelines
- Interactive debugging of TUI issues

## Architecture

The testing framework provides:
1. **High-level Session API** - Simple methods like `SendKeys`, `WaitForText`, `Capture`
2. **Automatic Cleanup** - Tmux sessions are automatically cleaned up after tests
3. **Flexible Capture** - Get raw output with ANSI codes or cleaned text
4. **Interactive Debugging** - Attach to live sessions for manual testing
5. **Headless Testing** - Test BubbleTea models without a terminal

This creates a "Playwright for Terminals" experience, enabling both automated testing and interactive debugging of TUI applications.