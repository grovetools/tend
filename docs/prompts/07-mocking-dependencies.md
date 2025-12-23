# Instructions for: Mocking and Dependency Management

Explain the mocking system and the ability to swap in real dependencies. Adhere to the style guide.

**Content Outline:**

1.  **Mocking with `harness.SetupMocks`:**
    *   Explain that `SetupMocks` is a `Step` that prepares the test environment.
    *   Describe how it creates a sandboxed `bin` directory and symlinks mock binaries into it.
    *   Explain the convention: `harness.Mock{CommandName: "git"}` will look for a compiled binary at a conventional path (e.g., `tests/e2e/tend/mocks/bin/mock-git`).
    *   Show a code example of setting up a `git` mock.

2.  **Using Real Dependencies (`--use-real-deps`):**
    *   Explain the `--use-real-deps` flag for `tend run`.
    *   Describe its purpose: to swap a specific mock (or all mocks) with its real counterpart for integration testing.
    *   Explain the mechanism: it uses `grove dev current <tool>` to find the real binary path.
