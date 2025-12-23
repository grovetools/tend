# Instructions for: How Tend Works - The Proxy Architecture

Explain the architecture of `tend` and how it builds project-specific test binaries. Adhere to the style guide.

**Tone**: Focus on the mechanical details of the build and execution flow, not on setup instructions.

**Content Outline:**

1.  **The Proxy Binary Concept:**
    *   Explain that the `tend` binary installed in your PATH is a proxy, not the actual test runner.
    *   Describe the execution flow:
        1. When you run `tend run` (or any `tend` command) in a project directory, the proxy binary detects it's in a project context.
        2. It searches for the project's `grove.yml` configuration file to locate the project root.
        3. It looks for test sources in the conventional location: `tests/e2e/`.
        4. It compiles those test sources into a temporary, project-specific binary (typically `bin/tend-e2e`).
        5. It re-executes itself by replacing its process with the newly-built project binary using `exec.Command`.
        6. The project-specific binary now runs with all your scenarios loaded, handling the original command (e.g., `run`, `list`, `tui`).

2.  **Why This Architecture:**
    *   Explain the benefits of this approach:
        *   Each project's tests are self-contained. No central registry of scenarios.
        *   The project binary has direct access to its own scenarios, internal packages, and test helpers.
        *   `tend list` and `tend run` automatically discover scenarios because they're compiled into the binary.
        *   Test dependencies are isolated to each project's `go.mod`.

3.  **The Test Entrypoint Structure:**
    *   Describe the conventional `tests/e2e/main.go` file structure.
    *   Explain that this file must:
        1. Import all scenario packages.
        2. Collect all `*harness.Scenario` values into a slice.
        3. Call `app.Execute(scenarios)` to hand control to the tend framework.
    *   Provide a minimal code example showing the entrypoint pattern.
    *   Note that the `app.Execute` function is what provides the CLI commands (`run`, `list`, etc.) - your project doesn't implement those.

4.  **Build Process Details:**
    *   Explain that `tend` uses `go build` to compile the test binary.
    *   Mention that it compiles to `bin/tend-e2e` (or `bin/<project-name>-e2e`).
    *   State that the build happens automatically and is cached until test sources change.
    *   Note that build errors are shown immediately with clear error messages.

5.  **Integration Points:**
    *   Briefly mention that projects typically add a `Makefile` target (e.g., `test-e2e`) that simply calls `tend run`.
    *   Explain that the `tend` proxy handles discovery and building, so the Makefile target is just a thin wrapper.
    *   Show a minimal example:
        ```makefile
        test-e2e:
            @tend run $(ARGS)
        ```
