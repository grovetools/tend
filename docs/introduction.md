# Grove Tend: A Go Library for E2E Testing

A Go library for creating powerful, scenario-based end-to-end testing frameworks. Grove Tend provides the essential building blocks to replace fragile, ad-hoc bash scripts with structured, maintainable, and easily debuggable Go code.

Designed with a **library-first philosophy**, Grove Tend empowers you to build a custom testing CLI tailored specifically to your project's needs. This approach keeps your test definitions and logic directly within your Go codebase, improving discoverability and maintainability.

The framework offers a comprehensive suite of features to streamline the entire E2E testing lifecycle:

-   **Scenario-Based Structure:** Organize tests logically with `Scenarios`, `Steps`, and a shared `Context`.
-   **First-Class Mocking:** Define mocks in Go, compile them as binaries, and seamlessly swap between mocked and real dependencies during test runs.
-   **Interactive Debugging:** Step through complex scenarios one-by-one or leverage the powerful debug mode with automatic tmux integration for an unparalleled debugging experience.
-   **Rich Helper Packages:** Utilize built-in helpers for filesystem, Git, command execution, and assertions to write robust tests quickly.
-   **CI/CD Integration:** Generate standard JUnit or JSON reports and benefit from automatic GitHub Actions annotations for seamless integration into your pipelines.

By combining the power of Go with a thoughtfully designed testing harness, Grove Tend transforms end-to-end testing from a chore into a core part of your development workflow.
