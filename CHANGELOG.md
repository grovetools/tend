## v0.3.1 (2025-09-26)

### Bug Fixes

* add changelog parsing to release

## v0.3.0 (2025-09-26)

A powerful new TUI testing framework, nicknamed "Playwright for Terminals," has been introduced (ac7c7ab). This framework enables robust testing of interactive command-line applications by automating tmux sessions. Initial enhancements focused on stability, replacing brittle `sleep` calls with intelligent waiting (df9e493) and resolving race conditions (50a9eb5). Subsequent updates added advanced features including conditional flow control, predicate-based navigation, and comprehensive session recording with HTML/JSON export for unparalleled debugging (4fb1276). The framework was further improved with direct filesystem verification helpers, allowing tests to reliably interact with and assert on file creation and content (e92984a).

The documentation system has been completely overhauled, migrating from a legacy generator to the ecosystem-standard `grove-docgen` (c068c3e, 4bcb777). This new system runs in an isolated environment to prevent interference with local checkouts (cd048ab) and uses a flexible YAML configuration. A new `tend docs` command has been added for easy access to the generated documentation (d93a54f), making project documentation more discoverable and maintainable.

### Features

- Implement TUI testing framework ("Playwright for Terminals") with tmux and headless BubbleTea support (ac7c7ab)
- Enhance TUI testing with intelligent waiting and navigation via `WaitForUIStable` and `NavigateToText` (df9e493)
- Add advanced TUI testing with conditional flows, predicate-based navigation, and session recording to HTML/JSON (4fb1276)
- Enhance TUI testing with filesystem verification helpers and reliable interactions (e92984a)
- Add automated, LLM-driven documentation generation system (961f53a)
- Isolate documentation generation process to prevent interference with local checkouts (cd048ab)
- Migrate to the ecosystem-standard `grove-docgen` for documentation (c068c3e)

### Bug Fixes

- Resolve TUI testing race conditions and improve command reliability (50a9eb5)

### Code Refactoring

- Improve documentation system by optimizing context usage and file organization (727277b)
- Transform docs generation from monolithic XML to a configurable JSON/YAML system (4bcb777)
- Adopt standardized documentation system with a `docs` CLI command and conventional file paths (d93a54f)

### Chores

- Update .gitignore to include go.work files and un-ignore CLAUDE.md (6b8dc1e)
- Remove obsolete generated documentation files (173fb9e)

### File Changes

```
 .gitignore                            |   6 +
 CLAUDE.md                             |  30 ++
 Makefile                              |  65 ++-
 docs/best-practices.md                |  58 +++
 docs/core-concepts.md                 | 108 ++++
 docs/docgen.config.yml                |  33 ++
 docs/docs.rules                       |  26 +
 docs/introduction.md                  |  15 +
 docs/prompts/best-practices.prompt.md |  27 +
 docs/prompts/core-concepts.prompt.md  |  29 ++
 docs/prompts/introduction.prompt.md   |  12 +
 docs/prompts/usage-patterns.prompt.md |  29 ++
 docs/usage-patterns.md                | 211 ++++++++
 examples/tmux-tui-test/README.md      |  84 ++++
 examples/tmux-tui-test/main.go        | 894 ++++++++++++++++++++++++++++++++++
 go.mod                                |  15 +-
 go.sum                                |  24 +
 internal/cmd/root.go                  |   2 +
 pkg/docs/docs.go                      |  13 +
 pkg/docs/docs.json                    |  41 ++
 pkg/harness/context.go                |  48 ++
 pkg/harness/harness.go                |  45 +-
 pkg/harness/ui.go                     |  46 +-
 pkg/teatest/teatest.go                |  57 +++
 pkg/tui/recording.go                  | 456 +++++++++++++++++
 pkg/tui/session.go                    | 473 ++++++++++++++++++
 pkg/tui/session_test.go               | 391 +++++++++++++++
 27 files changed, 3224 insertions(+), 14 deletions(-)
```

## v0.2.20 (2025-09-16)

### Features

* add LocalOnly and ExplicitOnly scenario execution controls

## v0.2.19 (2025-09-13)

### Chores

* update Grove dependencies to latest versions

## v0.2.18 (2025-09-12)

### Chores

* rm indirect deps

## v0.2.17 (2025-09-04)

### Bug Fixes

* prevent real grove-hooks from being called during E2E tests

### Code Refactoring

* improve grove-tend mocking framework

## v0.2.16 (2025-08-29)

### Features

* add first-class mocking support to grove-tend

## v0.2.15 (2025-08-28)

### Chores

* add Grove ecosystem files

## v0.2.14 (2025-08-27)

### Chores

* update readme

## v0.2.13 (2025-08-26)

### Features

* **run:** execute all scenarios and display results in table format
* **command:** add background process execution capability

## v0.2.12 (2025-08-25)

### Continuous Integration

* add Git LFS disable to release workflow

## v0.2.11 (2025-08-25)

### Continuous Integration

* disable Git LFS and linting in workflow

## v0.2.10 (2025-08-25)

### Features

* auto-find project-specific tend binaries

## v0.2.9 (2025-08-25)

### Chores

* bump dependencies

### Features

* add tmux integration for interactive debugging

## v0.2.8 (2025-08-15)

### Continuous Integration

* switch to Linux runners to reduce costs

### Chores

* bump deps

### Bug Fixes

* use full binary path from grove.yml in harness

## v0.2.7 (2025-08-13)

### Features

* add centralized binary discovery via grove.yml
* add ci.yml

### Continuous Integration

* consolidate to single test job on macOS
* reduce test matrix to macOS with Go 1.24.4 only

## v0.2.6 (2025-08-12)

### Features

* implemented Table Format for tend list Command

### Chores

* **deps:** bump dependencies

## v0.2.5 (2025-08-08)

### Bug Fixes

* make harness a public pkg

## v0.2.4 (2025-08-08)

### Code Refactoring

* Transform grove-tend into a pure testing library

### Chores

* **deps:** bump dependencies

## v0.2.3 (2025-08-08)

### Features

* Add version command and build-time version injection

