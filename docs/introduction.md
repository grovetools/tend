# Grove Tend: A Go Library for E2E Testing

Grove Tend is a Go library for creating powerful, scenario-based end-to-end testing frameworks. It provides the essential building blocks to replace fragile, ad-hoc bash scripts with structured, maintainable, and easily debuggable Go code.

Designed with a **library-first philosophy**, Grove Tend empowers you to build a custom testing CLI tailored specifically to your project's needs. This approach keeps your test definitions and logic directly within your Go codebase, improving discoverability and maintainability while leveraging the full power of the Go language.

The framework offers a comprehensive suite of features to streamline the entire E2E testing lifecycle:

-   **Scenario-Based Structure:** Organize tests logically into `Scenarios` and `Steps` that share a common `Context`, making complex test flows easy to read and manage.
-   **First-Class Mocking:** Define mocks as robust Go binaries, moving beyond brittle shell scripts. Seamlessly swap between these mocks and their real counterparts using the `--use-real-deps` flag, enabling a smooth transition from component to full integration testing.
-   **Interactive Debugging:** Step through complex scenarios one-by-one with interactive mode (`-i`), or leverage the powerful debug mode (`-d`) for automatic `tmux` integration. This provides an unparalleled "Playwright for the terminal" debugging experience, allowing you to inspect state at any point.
-   **Rich Helper Packages:** Utilize built-in helpers for the filesystem (`fs`), Git (`git`), command execution (`command`), and assertions (`assert`) to write robust tests quickly and reliably.
-   **CI/CD Integration:** Generate standard JUnit or JSON reports and benefit from automatic GitHub Actions annotations for seamless integration into your pipelines. Control test execution in CI with `LocalOnly` and `ExplicitOnly` scenario flags.

By combining the power of Go with a thoughtfully designed testing harness, Grove Tend transforms end-to-end testing from a chore into a core part of your development workflow.