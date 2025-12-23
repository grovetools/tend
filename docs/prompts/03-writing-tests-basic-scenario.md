# Instructions for: Writing Tests: A Basic Scenario

Provide a complete, practical example of a simple test scenario. Adhere to the style guide.

**Content Outline:**

1.  **Scenario Goal:**
    *   State the goal of the example: to test a CLI tool that writes to a file.

2.  **The `main_test.go` Entrypoint:**
    *   Show a minimal `main_test.go` or `tests/e2e/main.go` file that collects and runs scenarios. Explain that this is the entrypoint for the project-specific test runner.

3.  **The Scenario File:**
    *   Show the complete Go code for a scenario that:
        *   Defines a `Scenario` using `harness.NewScenario`.
        *   Has a `Step` that uses `ctx.Bin()` to run the application under test.
        *   Has another `Step` that uses `fs.AssertContains` to verify the output file's content.
    *   Use code comments to explain each part of the scenario.
