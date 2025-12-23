# Instructions for: Writing Tests: Assertions

Explain the two assertion styles in `tend`. Base the explanation on `tests/e2e/scenarios_assertions.go` and the `tend-tester.md` agent prompt. Adhere to the style guide.

**Content Outline:**

1.  **Overview:**
    *   State that `tend` provides two assertion styles for different use cases.

2.  **Hard Assertions (`ctx.Check`)**:
    *   Explain that `ctx.Check` is for fail-fast assertions.
    *   Describe its use case: critical checks where subsequent steps are invalid if the assertion fails (e.g., preconditions, file existence before use).
    *   Provide a code example using `ctx.Check` with an `fs.AssertExists` call.

3.  **Soft Assertions (`ctx.Verify`)**:
    *   Explain that `ctx.Verify` collects all failures within its block and reports them together.
    *   Describe its use case: validating multiple independent properties of a single state (e.g., fields in command output, multiple file contents).
    *   Provide a code example using `ctx.Verify` to check multiple substrings in a command's output.

4.  **Writing Good Assertion Descriptions:**
    *   Explain that the first argument to `Check` and the assertion methods in `Verify` is a description.
    *   State the purpose of the description: to provide clear context in test reports.
    *   Provide examples of good vs. bad descriptions.
