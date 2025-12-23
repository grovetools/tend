# Instructions for: Introduction to grove-tend

Write an introduction to `grove-tend`, adhering strictly to the provided style guide.

**Tone**: Factual, descriptive, modest.
**Audience**: Senior engineers.
**Goal**: Explain what `tend` is and the problem it solves at a mechanical level.

**Content Outline:**

1.  **Primary Definition:**
    *   Define `grove-tend` as a Go library and CLI for writing structured, scenario-based end-to-end tests for command-line tools and TUIs.
    *   State its primary purpose: to replace collections of shell scripts with maintainable, hermetic Go tests.

2.  **Core Mechanics:**
    *   Explain that tests are written in Go as `Scenarios` composed of `Steps`.
    *   Mention that the `harness` executes these scenarios, managing state, sandboxed filesystems, and cleanup.
    *   Describe its integration model: `tend` acts as a proxy that builds and runs a project-specific test binary, which includes that project's test scenarios.

3.  **Key Capabilities (Bulleted List):**
    *   List the main features factually.
    *   Example: "Hermetic test execution via temporary, sandboxed filesystems and home directories."
    *   Example: "Programmatic control and state assertion for Terminal User Interfaces (TUIs) via tmux sessions."
    *   Example: "Interactive debugging modes for step-through execution and live TUI exploration."
