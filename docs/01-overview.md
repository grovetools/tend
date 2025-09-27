# Grove-Tend: End-to-End Testing Framework

The `grove-tend` repository provides a Go library for creating powerful, scenario-based end-to-end testing frameworks. It offers essential building blocks to replace fragile, ad-hoc bash scripts with structured, maintainable, and easily debuggable Go code.

## Core Philosophy: Library-First Testing Framework

Grove-Tend is built on a library-first philosophy that empowers you to build custom testing CLIs tailored specifically to your project's needs. Instead of providing a one-size-fits-all testing tool, it gives you the components to create testing frameworks that integrate naturally with your codebase and development workflow.

The framework serves developers who need sophisticated end-to-end testing capabilities while maintaining the flexibility and power of Go for complex testing scenarios.

## Dual Role: Testing Library and Development Tool

The `grove-tend` framework serves as both a comprehensive testing library providing essential testing primitives and a development tool that transforms end-to-end testing from a chore into a core part of your workflow.

The framework offers a comprehensive suite of features to streamline the entire E2E testing lifecycle:

-   **Scenario-Based Structure:** Organize tests logically into `Scenarios` and `Steps` that share a common `Context`, making complex test flows easy to read and manage.
-   **First-Class Mocking:** Define mocks as robust Go binaries, moving beyond brittle shell scripts. Seamlessly swap between these mocks and their real counterparts using the `--use-real-deps` flag, enabling a smooth transition from component to full integration testing.
-   **Interactive Debugging:** Step through complex scenarios one-by-one with interactive mode (`-i`), or leverage the powerful debug mode (`-d`) for automatic `tmux` integration. This provides an unparalleled "Playwright for the terminal" debugging experience, allowing you to inspect state at any point.
-   **Rich Helper Packages:** Utilize built-in helpers for the filesystem (`fs`), Git (`git`), command execution (`command`), and assertions (`assert`) to write robust tests quickly and reliably.
-   **CI/CD Integration:** Generate standard JUnit or JSON reports and benefit from automatic GitHub Actions annotations for seamless integration into your pipelines. Control test execution in CI with `LocalOnly` and `ExplicitOnly` scenario flags.

By combining the power of Go with a thoughtfully designed testing harness, Grove Tend transforms end-to-end testing from a chore into a core part of your development workflow.