Generate a comprehensive overview for `grove-tend`.

## Requirements
Based on the provided `README.md` and existing documentation, create an overview that covers:

1. **High-level description**: Explain that `grove-tend` is a Go library for creating scenario-based end-to-end testing frameworks. Emphasize its "library-first" design, where developers build a custom test runner for their project.

2. **Animated GIF placeholder**: Include `<!-- placeholder for animated gif -->`

3. **Key Features**: Detail the main features, including:
   - Scenario-based testing (`Scenario`, `Step`, `Context`)
   - A rich set of helper packages (`fs`, `git`, `command`, `assert`)
   - First-class mocking support with Go binaries
   - Advanced TUI testing ("Playwright for the Terminal") using `tmux`
   - Interactive debugging modes (`-i`, `-d`)

4. Ideal for:

CLI tool testing - Commands with file I/O, environment variables, exit codes
Integration testing - Tools that orchestrate other CLI programs (git, docker, kubectl)
TUI applications - Interactive terminal UIs with complex navigation
Workflow validation - Multi-step processes requiring state persistence
LLM-generated test suites - Structure optimized for AI generation/maintenance

6. **How it works**: Technical description of the architecture and workflow

7. **Role in the Grove Ecosystem**: Explain that `tend` is the standard for E2E testing across all Grove CLI tools, providing a consistent and robust way to validate functionality

8. **LLM-First Design Philosophy**: Emphasize that while tend tests may appear verbose and cumbersome to write by hand, they are specifically designed to be generated and maintained by LLMs. This design choice enables comprehensive test coverage that would be impractical with traditional manual test writing

9. **Interactive Debugging**: Highlight how the debug features (`-i`, `-d` flags) allow users to step through scenarios interactively, making it easy to understand test failures and iterate on fixes. `-d` will open new tmux panes set to execution tmp dir. Users can watch scenarios progress and observe what's happening. 

10. **Installation**: Include brief installation instructions at the bottom

## Installation Format
Include this condensed installation section at the bottom:

### Installation

Grove-tend is a Go library. Add it to your project:
```bash
go get github.com/mattsolo1/grove-tend
```

For CLI usage, install via Grove meta-CLI:
```bash
grove install tend
```

Verify CLI installation:
```bash
tend version
```

See the [Grove Installation Guide](https://github.com/mattsolo1/grove-meta/blob/main/docs/02-installation.md) for setup.

## Context
Grove-tend is a Go library for creating powerful, scenario-based end-to-end testing frameworks, with CLI tools for managing and running tests within Grove ecosystem projects.
