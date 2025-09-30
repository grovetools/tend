Generate a "Conventions" document for integrating `grove-tend` into a project.

## Requirements
Analyze the `Makefile`s from various Grove projects to document the standard conventions for setting up and running `tend` E2E tests.

### Topics to Cover:

1. **The Test Runner Binary**:
   - Explain that each project using `tend` creates its own test runner binary (e.g., `tests/e2e/main.go`)
   - Describe the purpose of this binary: to import `grove-tend` as a library and register project-specific `harness.Scenario`s
   - Point to `grove-context/tests/e2e/main.go` as a typical example
   - Describe how the global `tend` binary automatically located the `./bin/tend` binary and runs it

2. **Makefile Integration**:
   - Describe the standard set of `make` targets used across the ecosystem
   - **`test-e2e-build` / `test-tend-build`**: Explain that this target is responsible for compiling the project's custom test runner binary (from `tests/e2e/main.go`)
   - **`build-mocks`**: Explain that this target compiles any Go-based mock binaries (from `tests/mocks/` or similar) needed for the tests
   - **`test-e2e`**: Explain that this is the main entry point. Document the typical dependency chain: `test-e2e: build build-mocks test-e2e-build`. Then, show how it executes the compiled test runner (e.g., `$(BIN_DIR)/$(E2E_BINARY_NAME) run $(ARGS)`)
   - **`ARGS` Variable**: Explain that the `ARGS` variable is the standard way to pass arguments (like scenario names or flags) to the test runner from the command line (e.g., `make test-e2e ARGS="run -i my-scenario"`)

3. **Project Structure**:
   - Document the standard directory layout for tend tests:
     - `tests/e2e/` - E2E test runner and scenarios
     - `tests/mocks/` - Mock binaries for external dependencies
     - `bin/` - Compiled test runner binary location
   - Explain the naming conventions for test files and scenario functions

4. **LLM-Friendly Patterns**:
   - Emphasize that the verbose nature of tend tests is intentional for LLM comprehension
   - Document how scenario descriptions should clearly state their purpose
   - Explain how step names should be descriptive action phrases
   - Note that tests serve as living documentation of expected behavior

5. **Development Workflow**:
   - Document the typical development cycle with tend:
     - Write/generate tests with LLM assistance
     - Run tests with `make test-e2e`
     - Debug failures with `make test-e2e ARGS="-i"`
     - Iterate on implementation until tests pass
   - Explain how tend integrates with CI/CD pipelines

## Context Files to Read
- `Makefile` (root Makefile)
- `grove-context/Makefile`
- `grove-flow/Makefile`
- `grove-meta/Makefile`
- `grove-hooks/Makefile`
- `grove-claude-logs/Makefile`
