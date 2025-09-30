Generate a detailed "Examples" document for `grove-tend`.

## Requirements
Create three distinct, in-depth examples that illustrate the primary use cases of `tend` as seen across the Grove ecosystem. For each example, provide a conceptual overview, a simplified code snippet, and an explanation of the key concepts being demonstrated.

### Example 1: Basic CLI Testing
- **Goal**: Show how to test a standard command-line tool that interacts with the filesystem
- **Source Code**: Use `grove-claude-logs/tests/e2e/scenarios.go` as a reference
- **Concepts to Explain**:
    - Setting up a temporary file structure using `fs.WriteString` and `ctx.NewDir`
    - Executing the tool's binary using `command.New(...).Run()`
    - Verifying text and JSON output using `assert.Contains` and `json.Unmarshal`
    - Passing state between steps using `ctx.Set` and `ctx.GetString`

### Example 2: Integration Testing with Mocks
- **Goal**: Explain how to test a tool that depends on other command-line programs (like `git` or `docker`) using `tend`'s mocking framework
- **Source Code**: Use `grove-tend/examples/mocking-demo/main.go` as the primary reference, particularly the `GitWorkflowScenario`
- **Concepts to Explain**:
    - The `harness.SetupMocks` step and the `harness.Mock` struct
    - The convention of creating mock binaries in Go (e.g., `tests/mocks/git/main.go`)
    - The importance of using `ctx.Command(...)` to ensure the mock-aware `PATH` is used
    - How to swap mocks for real dependencies using the `--use-real-deps` flag for integration testing

### Example 3: TUI Testing (Experimental)
- **Goal**: Demonstrate how to test interactive Terminal User Interfaces (TUIs)
- **Source Code**: Use `grove-context/tests/e2e/scenarios_tui.go` as the reference
- **Warning**: Mark as experimental feature
- **Concepts to Explain**:
    - Launching a TUI in an isolated `tmux` session with `ctx.StartTUI`
    - The `tui.Session` object
    - Interacting with the TUI using `session.SendKeys`
    - Waiting for the UI to be in a specific state using `session.WaitForText`, `session.WaitForUIStable`, and `session.WaitForAnyText`
    - Asserting on screen content with `session.AssertContains`
    - Advanced interaction with `session.SelectItem` and `session.NavigateToText`
    - The interactive debugging workflow (`-d` flag and attaching to the `tmux` session)

## Additional Context
- Highlight that tend tests are designed to be written and maintained by LLMs
- Show how the verbose nature of tend tests provides excellent documentation of system behavior
