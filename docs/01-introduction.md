`grove-tend` is a Go library and command-line tool for writing structured, scenario-based end-to-end tests for command-line interfaces and Terminal User Interfaces (TUIs). Its purpose is to replace collections of shell scripts with maintainable, hermetic tests written in Go.

Tests are defined as `Scenarios` composed of sequential `Steps`. The test `harness` executes these scenarios, managing state, sandboxed filesystems, mock dependencies, and automatic cleanup.

When executed, the `tend` CLI acts as a proxy. It discovers the project under test, builds a project-specific test binary containing that project's compiled `Scenario` definitions, and then executes that binary with the specified arguments.

Key capabilities include:

*   Hermetic test execution via temporary, sandboxed filesystems and home directories.
*   Mocking of command-line dependencies (e.g., `git`, `docker`, `kubectl`).
*   Programmatic control and state assertion for TUIs via managed tmux sessions.
*   Helpers for manipulating Git repositories, running commands, and managing Docker containers.
*   Interactive debugging modes for step-through execution and live TUI exploration.