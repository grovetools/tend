# Examples

## Example 1: Basic CLI Testing

This example demonstrates how to test a command-line tool that interacts with the filesystem. The scenario tests a `clogs list` command that reads files from a directory and outputs JSON.

### Overview

The test verifies that `clogs list --json` correctly reads files from a mock `~/.claude` directory, parses their content, and displays the output in JSON format. The test creates a temporary, isolated filesystem, runs the command against it, and asserts that the command's output is correct.

### Code Example

```go
// From: grove-claude-logs/tests/e2e/scenarios.go

// setupMockClaudeDir creates a temporary ~/.claude directory for the test.
func setupMockClaudeDir(ctx *harness.Context) error {
    // Create a temporary "home" directory managed by the harness.
    homeDir := ctx.NewDir("home")

    // Create the required directory structure.
    projectsDir := filepath.Join(homeDir, ".claude", "projects", "test-project")
    if err := fs.CreateDir(projectsDir); err != nil {
        return err
    }

    // Write mock session files.
    transcriptContent := `{"sessionId":"session-alpha","message":{"role":"user","content":"Hello"}}`
    if err := fs.WriteString(filepath.Join(projectsDir, "session-alpha.jsonl"), transcriptContent); err != nil {
        return err
    }

    // Store the path to the temporary home directory in the context for later steps.
    ctx.Set("mock_home", homeDir)
    return nil
}

// ClogsListScenario tests the 'clogs list' command.
func ClogsListScenario() *harness.Scenario {
    return &harness.Scenario{
        Name: "clogs-list-command",
        Steps: []harness.Step{
            harness.NewStep("Setup mock Claude directory", setupMockClaudeDir),
            harness.NewStep("Run 'clogs list --json'", func(ctx *harness.Context) error {
                // Find the binary under test.
                clogsBinary, err := FindProjectBinary()
                if err != nil {
                    return err
                }

                // Retrieve the temporary home directory path from the context.
                homeDir := ctx.GetString("mock_home")

                // Execute the command, overriding the HOME environment variable.
                cmd := command.New(clogsBinary, "list", "--json").Env("HOME=" + homeDir)
                result := cmd.Run()
                ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)

                if result.ExitCode != 0 {
                    return fmt.Errorf("clogs list --json failed: %s", result.Stderr)
                }

                // Verify the output is valid JSON and contains the expected data.
                var sessions []map[string]interface{}
                if err := json.Unmarshal([]byte(result.Stdout), &sessions); err != nil {
                    return fmt.Errorf("failed to parse JSON output: %w", err)
                }

                if len(sessions) == 0 {
                    return fmt.Errorf("expected at least one session in JSON output")
                }

                // Assert that a specific session is present.
                found := false
                for _, s := range sessions {
                    if id, ok := s["sessionId"].(string); ok && id == "session-alpha" {
                        found = true
                        break
                    }
                }
                if !found {
                    return fmt.Errorf("session-alpha not found in JSON output")
                }

                return nil
            }),
        },
    }
}
```

### Key Concepts

*   **Filesystem Setup**: The `fs` helper package is used to create an isolated test environment. `ctx.NewDir("home")` creates a temporary directory managed by the harness, and `fs.WriteString` populates it with mock data. This ensures the test does not depend on or interfere with the local filesystem.
*   **Command Execution**: The `command.New()` function creates a new command to be executed. The `.Env()` method allows for overriding environment variables, which is critical for redirecting the CLI tool to use the temporary test directory. The `.Run()` method executes the command and returns a `Result` struct containing stdout, stderr, and the exit code.
*   **State Management**: The `ctx.Set("mock_home", homeDir)` call stores the path of the temporary home directory in the scenario's context. A later step retrieves this path using `ctx.GetString("mock_home")`, enabling state to be passed reliably between steps.
*   **Verification**: The `assert` package provides functions for validation. In this example, `json.Unmarshal` is used to verify that the command's output is valid JSON, and further checks confirm that the data contains the expected session information.

## Example 2: Integration Testing with Mocks

This example shows how to test a tool that depends on other command-line programs, such as `git` or `docker`. The `tend` framework provides a mocking system that uses compiled Go binaries.

### Overview

The test simulates a Git workflow (`init`, `status`, `add`, `commit`) to verify that a tool interacts with `git` correctly. Instead of using the actual `git` binary, the test uses a mock implementation written in Go. This allows for fast, predictable, and isolated testing without requiring `git` to be installed or interacting with a real repository.

### Code Example

```go
// From: grove-tend/examples/mocking-demo/main.go

// This scenario demonstrates mocking the 'git' command.
var GitWorkflowScenario = &harness.Scenario{
    Name: "git-workflow",
    Tags: []string{"git", "mocking"},
    Steps: []harness.Step{
        // This step sets up the mocks for the scenario.
        harness.SetupMocks(
            harness.Mock{CommandName: "git"},
        ),

        harness.NewStep("Initialize git repository", func(ctx *harness.Context) error {
            repoDir := ctx.NewDir("repo")
            ctx.Set("repo_dir", repoDir)
            
            // ctx.Command ensures the command runs with the mock-aware PATH.
            cmd := ctx.Command("git", "init").Dir(repoDir)
            result := cmd.Run()
            ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
            
            if result.Error != nil {
                return result.Error
            }
            return assert.Contains(result.Stdout, "Initialized empty Git repository")
        }),

        harness.NewStep("Stage and commit files", func(ctx *harness.Context) error {
            repoDir := ctx.GetString("repo_dir")
            
            // These commands will execute the mock git binary.
            addCmd := ctx.Command("git", "add", ".").Dir(repoDir)
            if err := addCmd.Run().Error; err != nil {
                return fmt.Errorf("git add failed: %w", err)
            }
            
            commitCmd := ctx.Command("git", "commit", "-m", "Initial commit").Dir(repoDir)
            result := commitCmd.Run()
            ctx.ShowCommandOutput(commitCmd.String(), result.Stdout, result.Stderr)
            
            return assert.Contains(result.Stdout, "Initial commit", "mock commit output")
        }),
    },
}
```

### Key Concepts

*   **`harness.SetupMocks`**: This is a step builder that prepares the test environment for mocking. For each `harness.Mock{CommandName: "git"}`, it finds a pre-compiled mock binary (e.g., `bin/mock-git`) and creates a symlink named `git` inside a temporary `bin` directory. This directory is then prepended to the `PATH` for the duration of the scenario.
*   **Go-based Mocks**: The mock for `git` is a Go program (e.g., `tests/mocks/git/main.go`). This allows for more complex and stateful mock behavior than shell scripts.
*   **`ctx.Command(...)`**: This is a mock-aware factory for creating commands. When `ctx.Command("git", "init")` is used, `tend` ensures that the `git` executable resolves to the mock binary symlinked in the temporary `bin` directory, rather than the real `git` on the system `PATH`.
*   **`--use-real-deps` Flag**: `tend` allows for swapping mocks with real binaries for integration testing. Running the test with `./my-tests run git-workflow --use-real-deps=git` instructs the `SetupMocks` step to symlink the actual `git` binary (found via `grove dev current git`) instead of the mock. This provides a way to switch from component tests to full integration tests.

### Automatic PATH Handling for TUI Sessions

When testing TUI applications that call external commands, the framework automatically handles PATH manipulation for mock binaries. Simply set the `test_bin_dir` context key with your mock directory:

```go
// Create mocks
mockDir := ctx.NewDir("mocks")
mockGitPath := filepath.Join(mockDir, "git")
fs.WriteString(mockGitPath, mockGitScript)
os.Chmod(mockGitPath, 0755)

// Set the convention key - StartTUI will automatically prepend this to PATH
ctx.Set("test_bin_dir", mockDir)

// Launch TUI - PATH is automatically configured!
session, err := ctx.StartTUI(myBinary, []string{"arg1", "arg2"})
```

This eliminates the need for wrapper scripts or manual PATH manipulation. The framework automatically:
- Detects the `test_bin_dir` context key
- Prepends it to PATH when launching TUI sessions
- Merges with any user-provided environment variables via `tui.WithEnv()`

See `examples/auto-path-mocks` for a complete demonstration.

## Example 3: TUI Testing (Experimental)

> **Warning:** TUI testing is an experimental feature. Its API and behavior may change.

The `tend` framework includes capabilities for testing interactive Terminal User Interfaces (TUIs). It automates `tmux` sessions to run a TUI in an isolated environment, allowing the test to send keystrokes and assert on the screen content.

### Overview

This test launches a `cx view` TUI session, which displays a file tree. The scenario verifies that the TUI starts correctly, then navigates the file list using keystrokes, and finally asserts that the screen content updates as expected. The entire process is headless and automated.

### Code Example

```go
// From: grove-context/tests/e2e/scenarios_tui.go

func TUIViewScenario() *harness.Scenario {
    return &harness.Scenario{
        Name: "cx-view-tui-test",
        Tags: []string{"cx", "tui", "view"},
        Steps: []harness.Step{
            harness.NewStep("Setup project files for TUI", func(ctx *harness.Context) error {
                fs.WriteString(filepath.Join(ctx.RootDir, "main.go"), "package main")
                fs.WriteString(filepath.Join(ctx.RootDir, "README.md"), "# Project")
                rules := "**/*.go"
                return fs.WriteString(filepath.Join(ctx.RootDir, ".grove", "rules"), rules)
            }),
            harness.NewStep("Launch 'cx view' in tmux", func(ctx *harness.Context) error {
                cxBinary, _ := FindProjectBinary()

                // Launch the TUI in an isolated, managed tmux session.
                session, err := ctx.StartTUI(cxBinary, "view")
                if err != nil {
                    return fmt.Errorf("failed to start TUI: %w", err)
                }
                
                // Store the session handle for later steps.
                ctx.Set("view_session", session)
                return nil
            }),
            harness.NewStep("Wait for TUI to stabilize and verify content", func(ctx *harness.Context) error {
                session := ctx.Get("view_session").(*tui.Session)
                
                // Wait for the UI to stop changing, which is more reliable than a fixed sleep.
                if err := session.WaitForUIStable(5*time.Second, 100*time.Millisecond, 300*time.Millisecond); err != nil {
                    return fmt.Errorf("TUI did not stabilize: %w", err)
                }

                // Wait for specific text to ensure the file tree has loaded.
                if err := session.WaitForText("main.go", 2*time.Second); err != nil {
                    return fmt.Errorf("main.go not found in UI: %w", err)
                }
                
                // Assert that the file has the correct status indicator.
                return session.AssertContains("✓ main.go")
            }),
            harness.NewStep("Navigate and interact with the TUI", func(ctx *harness.Context) error {
                session := ctx.Get("view_session").(*tui.Session)

                // Use a predicate to reliably select a specific item in the list.
                err := session.SelectItem(func(line string) bool {
                    return strings.Contains(line, "README.md")
                })
                if err != nil {
                    return fmt.Errorf("failed to select README.md: %w", err)
                }

                // Send the 'h' key to add the item to the hot context.
                return session.SendKeys("h")
            }),
            harness.NewStep("Quit the TUI", func(ctx *harness.Context) error {
                session := ctx.Get("view_session").(*tui.Session)
                return session.SendKeys("q")
            }),
        },
    }
}
```

### Key Concepts

*   **`ctx.StartTUI`**: This function launches the specified command in a new, isolated `tmux` session. It returns a `*tui.Session` object, which is the primary handle for interacting with the TUI. The harness automatically manages the lifecycle of this `tmux` session.
*   **`tui.Session`**: This object provides an API for TUI interaction.
    *   `SendKeys(...)`: Sends keystrokes to the TUI (e.g., `"h"`, `"q"`, `"Enter"`, `"Down"`).
    *   `WaitForText(...)`: Polls the screen content until a specific string appears.
    *   `WaitForUIStable(...)`: Waits until the screen content stops changing for a specified duration, useful for animations or asynchronous loading.
    *   `WaitForAnyText(...)`: Waits for one of several possible text strings to appear, useful for conditional UI outcomes.
    *   `AssertContains(...)`: Immediately checks if the screen content includes a specific string.
    *   `SelectItem(...)`: Finds a line matching a predicate, moves the cursor to it, and presses `Enter`.
    *   `NavigateToText(...)`: Moves the cursor from its current position to a target string on the screen by sending the necessary arrow key presses.
*   **Interactive Debugging (`-d` flag)**: When a TUI test is run with the `-d` flag, the test runner pauses before each step and offers an `(a)ttach` option. This allows the developer to attach directly to the `tmux` session, manually interact with the TUI to inspect its state, and then detach (`Ctrl-b d`) to resume the automated test. This provides a workflow for debugging TUI interactions.
