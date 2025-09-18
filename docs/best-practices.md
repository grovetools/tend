## Best Practices

### Use Descriptive and Consistent Naming

Scenario and Step names should be clear, concise, and descriptive. A good name immediately communicates the test's intent and makes failures easier to diagnose from logs and reports.

- **Scenario Names**: Use a consistent pattern like `feature-or-entity-action`. For example, `git-workflow` or `webapp-smoke-test`.
- **Step Names**: Describe the action being performed, such as `Initialize git repository` or `Verify file exists`.
- **Descriptions**: Use the `Description` field on both Scenarios and Steps to provide additional context that appears in the `tend list --verbose` output.

### Embrace Structured Error Handling

Every `Step.Func` should handle errors properly by returning them. The harness will automatically stop the scenario on the first non-nil error.

- **Wrap Errors**: Provide context when returning errors to make debugging easier. Use `fmt.Errorf` with the `%w` verb: `return fmt.Errorf("failed to write config: %w", err)`.
- **Use Assertions**: Leverage the `pkg/assert` package for readable and explicit checks. `return assert.Contains(result.Stdout, "expected output")` is clearer than manual string checks and provides better failure messages.

### Ensure Scenarios are Self-Contained and Isolated

Each scenario should be able to run independently without relying on the state left by others. The harness facilitates this by providing a unique, temporary directory for each scenario run, accessible via `ctx.RootDir`.

- **Use the Test Directory**: All test artifacts, files, and repositories should be created within `ctx.RootDir`.
- **Manage State with Context**: Pass data between steps using `ctx.Set()` and `ctx.Get()`. Avoid using global variables, which can create dependencies between tests and prevent parallel execution.

### Write Efficient and Performant Tests

Avoid fixed-length sleeps, as they make tests slow and flaky. Instead of `time.Sleep()` or `harness.DelayStep`, use polling mechanisms that wait for a specific condition to be met.

- **Use Wait Helpers**: The `pkg/wait` package provides functions like `wait.For`, `wait.ForHTTP`, and `wait.ForFileContent` that poll a condition until it succeeds or a timeout is reached. This makes tests both faster and more reliable.

### Structure Scenarios for Maintainability

Well-structured tests are easier to read, update, and debug.

- **Leverage Helper Packages**: Use the built-in helpers (`fs`, `git`, `command`, `assert`, `wait`) to perform common operations robustly.
- **Use Step Builders**: Use functions like `harness.NewStep` and `harness.ConditionalStep` to create clear, reusable steps.
- **Write Mocks in Go**: Instead of brittle shell scripts, write your mocks as Go programs. This makes them more powerful, stateful, and easier to maintain. Integrate them using the `harness.SetupMocks` step.

### Leverage In-Code Documentation

Both `harness.Scenario` and `harness.Step` have `Description` fields. Use them to document the purpose and intent of your tests. This information is displayed when using `tend list --verbose`, providing valuable context for other developers and making the test suite self-documenting.

### Design Scenarios for CI/CD

Tend provides features to help manage which tests run in different environments.

- **Local-Only Scenarios**: Mark scenarios that require a specific local setup or are not suitable for CI with `LocalOnly: true`. They will be automatically skipped in CI environments.
- **Explicit-Only Scenarios**: For expensive or long-running integration tests, use `ExplicitOnly: true`. These tests are skipped by default during a full `tend run` and must be invoked by name or with the `--explicit` flag.
- **Generate Reports**: Use the `--junit <file>` and `--json <file>` flags in your CI pipeline to generate machine-readable reports for test analytics and integration with other tools.

### Master Debugging Techniques

Tend offers powerful tools to troubleshoot failing tests.

- **Interactive Mode (`-i`):** Pause execution before each step, allowing you to inspect the state of the system.
- **No Cleanup (`--no-cleanup`):** Prevents the deletion of the temporary test directory (`ctx.RootDir`), so you can examine the generated files and logs after a run.
- **Verbose Output (`-v`, `--very-verbose`):** Increase the level of detail in the output. `--very-verbose` includes command output.
- **Debug Mode (`-d`):** The ultimate debugging tool. It's a shorthand for `-i --no-cleanup --very-verbose --tmux-split`, which runs the test interactively and automatically splits your `tmux` window, `cd`-ing into the test's temporary directory for hands-on inspection.

