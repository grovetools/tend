### Mocking and Dependency Management

The `tend` framework provides a system for mocking command-line dependencies, with the ability to selectively swap in real binaries for integration testing.

#### Mocking with `harness.SetupMocks`

The `harness.SetupMocks` function returns a `harness.Step` that prepares the test environment by creating a sandboxed `bin` directory for mock binaries. During a test run, this directory (`test_bin`) is created inside the test's root directory. `SetupMocks` then creates symbolic links to pre-compiled mock binaries for each specified command. When `ctx.Command()` is used in subsequent steps, its PATH is automatically configured to prioritize this directory, ensuring the mock is executed instead of the real tool.

By convention, `harness.Mock{CommandName: "git"}` looks for a compiled binary at one of the following paths relative to the project root:

-   `tests/e2e/tend/mocks/bin/mock-git`
-   `tests/mocks/bin/mock-git`
-   `bin/mock-git`

These mock binaries are typically built using a `make build-e2e-mocks` or `make build-mocks` target defined in the project's `Makefile`.

**Example:**

```go
// File: tests/e2e/scenarios_mocking.go
var GitWorkflowScenario = harness.NewScenario(
    "git-workflow",
    "Tests a git workflow using mocked git commands",
    []string{"git", "mocking"},
    []harness.Step{
        // This step sets up the mock environment.
        harness.SetupMocks(harness.Mock{CommandName: "git"}),

        harness.NewStep("Initialize git repository", func(ctx *harness.Context) error {
            // ctx.Command("git", "init") will now execute the mock binary.
            cmd := ctx.Command("git", "init").Dir(ctx.RootDir)
            result := cmd.Run()
            return result.AssertSuccess()
        }),
    },
)
```

#### Using Real Dependencies (`--use-real-deps`)

The `tend run` command includes a `--use-real-deps` flag to selectively replace mocks with their real counterparts for integration testing. When this flag is used, `tend` bypasses the mock symlink for the specified tool and instead locates the real binary by executing `grove dev current <tool-name>`. This allows a test to run against a real dependency without altering the scenario code.

You can specify a comma-separated list of tools to swap or use the special value `all` to replace every mock with its real equivalent.

**Examples:**

```bash
# Run all tests, but use the real `flow` binary instead of its mock
tend run --use-real-deps=flow

# Run a specific test using real `git` and `docker` binaries
tend run git-workflow --use-real-deps=git,docker

# Run tests using all available real dependencies instead of mocks
tend run --use-real-deps=all
```