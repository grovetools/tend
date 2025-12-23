# Instructions for: Writing Tests: Filesystem Operations

Explain how to interact with the sandboxed filesystem. Adhere to the style guide.

**Content Outline:**

1.  **Sandboxed Environment:**
    *   Explain that each scenario runs in its own temporary root directory.
    *   Describe the sandboxed home directory structure (`home/`, `.config/`, etc.) that is automatically created and configured via `HOME` and `XDG_*` environment variables.

2.  **Creating Directories (`ctx.NewDir`):**
    *   Explain how `ctx.NewDir("my-dir")` creates a directory within the test's root and returns its path.

3.  **Accessing Sandboxed Paths:**
    *   Explain `ctx.RootDir`, `ctx.HomeDir`, `ctx.ConfigDir`, etc.

4.  **Filesystem Helpers (`pkg/fs`):**
    *   Mention the helpers available in the `pkg/fs` package.
    *   Provide examples for `fs.WriteString` and `fs.AssertExists`.
