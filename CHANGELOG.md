## v0.5.0 (2026-01-14)

This release introduces a comprehensive suite of features for interactive test development, debugging, and execution, with special emphasis on TUI testing. Key additions include an interactive TUI for browsing and running tests, an advanced `--debug-session` mode for isolated development environments, and a parallel test runner. 

An proxy architecture now automatically builds project-specific test runners on demand (254a3f8), injecting version information (64065f0) and building mocks by convention (d1f25ca). This process is optimized to skip unnecessary rebuilds for commands like `version` or `sessions`, significantly improving performance (330e74f).

A new interactive terminal UI, accessible via `tend tui`, provides a convenient way to browse, filter, and run test scenarios across workspaces (e0f2cc3). 

The debugging workflow has been overhauled with the `--debug-session` flag, which creates an isolated, multi-window tmux environment for running tests, editing code, and inspecting artifacts simultaneously (7dca5a3, 71f56d8). A new `tend sessions` command and TUI provide capabilities to list, preview, attach to, and manage these debug sessions programmatically (1bccddf, d7f1ace).

The TUI testing framework itself has been significantly enhanced. A new `Type()` helper simplifies test interactions and handles vim-style chords (c6234f8, 8191af7). Reliability is improved with stable waiting helpers like `WaitStable()` (e29afba) and flexible assertions via `AssertLine()` (5b0c0a6). To aid in writing these tests, a new `tend record` command captures manual TUI interactions into interactive HTML, Markdown, and XML reports (cc91acb, ab41ad5), and failed TUI tests can be automatically recorded using the `--record-tui` flag (5b0c0a6).

For large-scale testing, a parallel test runner with its own TUI progress display is now available via `tend run --parallel` (2dc244c), which includes detailed failure reporting (6a0bc9e). Furthermore, the new `tend ecosystem run` command can discover and execute all test suites across the entire Grove ecosystem, presenting an aggregated summary and detailed failure analysis (b35b715, 0868ed9).

Writing tests is now easier with the introduction of formal setup and teardown lifecycle phases (c1b9d4c) and a suite of comprehensive assertion helpers for command outputs, files, and YAML content (02a6dba). The test harness now automatically sandboxes the environment by setting `HOME` and `XDG_*` variables (cfb5c98) and correctly detects the Docker socket (375310b), ensuring test isolation.

### BREAKING CHANGES

- `tend sessions capture` now strips ANSI codes from output by default. The new `--with-ansi` flag must be used to preserve them. (4b918b8)

### Features

- **Proxy & Build**: Automatically build project-specific test runners on demand, simplifying the development workflow. (254a3f8)
- **Proxy & Build**: Inject git version information into auto-built test runners. (64065f0)
- **Proxy & Build**: Optimize proxy performance by skipping rebuilds for commands that don't need project-specific runners. (330e74f)
- **Interactive TUI**: Implement `tend tui` for browsing, filtering, and running tests interactively from a hierarchical tree view. (e0f2cc3)
- **Interactive TUI**: Add search functionality with `/` to filter test scenarios. (920e660)
- **Interactive TUI**: Add a split-pane test runner with live output display, triggered by the 'r' key. (d7d81ba)
- **Interactive TUI**: Add `Ctrl+U`/`Ctrl+D` for half-page navigation. (3d8528f)
- **Interactive TUI**: Show indicators for `explicit-only` and `local-only` tests. (568511d)
- **Debugging**: Add `--debug-session` mode for an isolated, multi-window tmux debugging environment. (7dca5a3)
- **Debugging**: Enhance `--debug-session` with five specialized windows for running, editing, and inspecting tests. (71f56d8)
- **Debugging**: Add a debug editor view with automatic source code navigation to scenario and step definitions. (f2f1d43)
- **Debugging**: Add `--run-steps` flag to automatically run specific steps before pausing in interactive mode. (53b015b)
- **Session Management**: Add `tend sessions` command with a TUI for listing, previewing, and managing debug sessions. (1bccddf)
- **Session Management**: Add CLI subcommands to `tend sessions` (list, kill, capture, send-keys) for programmatic TUI exploration. (d7f1ace)
- **Session Management**: Add `--strip-ansi` and `--wait-for` flags to `tend sessions capture` for easier agent consumption. (cb658b8)
- **TUI Recording**: Add `tend record` command to capture manual TUI sessions into interactive HTML, Markdown, and XML reports. (cc91acb, ab41ad5, 5d539af)
- **TUI Testing**: Add `--record-tui` flag to `tend run` to automatically record TUI sessions for failed tests. (5b0c0a6)
- **TUI Testing**: Add `WaitStable()` helper for waiting for TUI content to stop changing. (e29afba)
- **TUI Testing**: Add `AssertLine()` for flexible, line-based assertions without relying on ANSI codes. (5b0c0a6)
- **TUI Testing**: Add `Type()` helper to simplify TUI interactions and intelligently handle vim-style chord commands. (c6234f8)
- **Parallel & Ecosystem Testing**: Implement a parallel test runner (`tend run --parallel`) with a TUI for real-time progress. (2dc244c)
- **Parallel & Ecosystem Testing**: Add detailed failure output for parallel test runs. (6a0bc9e)
- **Parallel & Ecosystem Testing**: Add `tend ecosystem run` command to discover and execute test suites across the ecosystem. (b35b715)
- **Parallel & Ecosystem Testing**: Add a split-pane TUI with streaming output for the ecosystem runner. (0868ed9)
- **Harness**: Add setup and teardown lifecycle phases to test scenarios. (c1b9d4c)
- **Harness**: Add comprehensive assertion helpers for command results, filesystem, and YAML files. (02a6dba)
- **Harness**: Implement automatic XDG config sandboxing for all tests. (cfb5c98)
- **Harness**: Add automatic Docker socket detection for sandboxed environments to prevent path length issues. (375310b)
- **Harness**: Isolate TUI tests in dedicated tmux servers to avoid disrupting user sessions. (d4d2bb5)
- **E2E Tests**: Implement a dedicated `tests/e2e` suite for `grove-tend` to dogfood the framework itself. (f67fadb)

### Bug Fixes

- Use per-pane environment variables in tmux to fix command truncation issues with fish shell. (61a8d33)
- Fix TUI bugs related to logging spam, duplicate scenario display, and incorrect test execution logic. (2712d93)
- Recover gracefully from stale tmux pane IDs when reusing debug panes across sessions. (2b111d4)
- Expand vim chord detection in `Type()` to handle all two-key sequences (e.g., "zM"). (8191af7)
- Improve TUI recorder to create one frame per keystroke for more accurate playback. (f530c73)
- Update parallel runner tests for improved reliability and correct binary usage. (64cdcec)
- Replace regex-based parsing with robust JSON-based parsing for test results in the ecosystem TUI. (680c1c9)
- Only show the proxy build message after confirming a project-specific test runner exists. (277a668)

### Refactoring

- Replace all emojis with Nerd Font icons from the `grove-core` theme for improved consistency and compatibility. (29a1745)
- Migrate all UI components to use the centralized theme system from `grove-core`. (d7f9e70)
- Rename the `--setup-only` flag to `--run-setup` for improved semantic clarity. (770d6e1)
- Migrate internal TUI tests to use the new simplified `Type()` API, reducing code verbosity. (247144d)
- Rename the `tend tui record` command to `tend record` as it can record any TUI application. (0dc7497)

### File Changes

```
 .cx/dev-no-tests.rules                             |   19 +
 .cx/dev-with-tests.rules                           |   14 +
 .cx/docs-only.rules                                |    8 +
 .github/workflows/ci.yml                           |    4 +
 .github/workflows/release.yml                      |   20 +-
 .gitignore                                         |    6 +-
 CHANGELOG.md                                       |    2 +
 Makefile                                           |   37 +-
 docs/01-introduction.md                            |   13 +
 docs/01-overview.md                                |   70 --
 docs/02-core-concepts.md                           |   69 ++
 docs/02-examples.md                                |  277 -----
 docs/03-conventions.md                             |  114 --
 docs/03-writing-tests-basic-scenario.md            |  106 ++
 docs/04-writing-tests-assertions.md                |   82 ++
 docs/05-writing-tests-filesystem.md                |   74 ++
 docs/06-writing-tests-commands.md                  |   44 +
 docs/07-mocking-dependencies.md                    |   56 +
 docs/08-tui-testing-basics.md                      |   84 ++
 docs/09-tui-testing-interaction.md                 |   79 ++
 docs/10-tui-testing-assertions.md                  |  108 ++
 docs/11-interactive-debugging.md                   |   59 +
 docs/12-cli-reference.md                           |   55 +
 docs/13-project-integration.md                     |   84 ++
 docs/README.md.tpl                                 |   18 +-
 docs/docgen.config.yml                             |  111 +-
 docs/docs.rules                                    |    5 +
 docs/images/grove-tend-readme.svg                  | 1192 --------------------
 docs/prompts/01-overview.md                        |   58 -
 docs/prompts/02-examples.md                        |   39 -
 docs/prompts/03-conventions.md                     |   48 -
 examples/auto-path-mocks/README.md                 |   90 --
 examples/auto-path-mocks/main.go                   |  195 ----
 examples/custom-tend/main.go                       |  691 ------------
 examples/env-test/main.go                          |  132 ---
 examples/env-test/simple-env-test.sh               |    4 -
 examples/env-test/test-env                         |  Bin 7982466 -> 0 bytes
 examples/mocking-demo/.gitignore                   |   11 -
 examples/mocking-demo/Makefile                     |  117 --
 examples/mocking-demo/README.md                    |  162 ---
 examples/mocking-demo/main.go                      |  315 ------
 examples/tmux-tui-test/README.md                   |   84 --
 examples/tmux-tui-test/main.go                     |  894 ---------------
 go.mod                                             |   15 +-
 go.sum                                             |   47 +-
 grove.yml                                          |    3 +
 internal/cmd/demo/demo.go                          |  143 ++-
 internal/cmd/ecosystem.go                          |  484 ++++++++
 internal/cmd/list.go                               |   96 +-
 internal/cmd/root.go                               |    6 +-
 internal/cmd/run.go                                |  482 +++++++-
 internal/cmd/sessions.go                           |  258 +++++
 internal/cmd/tui.go                                |  189 ++++
 internal/cmd/validate.go                           |    3 +-
 internal/tui/e_runner/delegate.go                  |   64 ++
 internal/tui/e_runner/io.go                        |   35 +
 internal/tui/e_runner/model.go                     |  115 ++
 internal/tui/e_runner/runner.go                    |  175 +++
 internal/tui/e_runner/update.go                    |  125 ++
 internal/tui/e_runner/view.go                      |   38 +
 internal/tui/prunner/delegate.go                   |   52 +
 internal/tui/prunner/io.go                         |   42 +
 internal/tui/prunner/model.go                      |  102 ++
 internal/tui/prunner/runner.go                     |  132 +++
 internal/tui/prunner/update.go                     |   83 ++
 internal/tui/prunner/view.go                       |   25 +
 internal/tui/runner/io.go                          |  343 ++++++
 internal/tui/runner/keymap.go                      |  101 ++
 internal/tui/runner/model.go                       |  117 ++
 internal/tui/runner/update.go                      |  691 ++++++++++++
 internal/tui/runner/view.go                        |  205 ++++
 internal/tui/scanner/scanner.go                    |  232 ++++
 internal/tui/sessions/io.go                        |  118 ++
 internal/tui/sessions/model.go                     |   74 ++
 internal/tui/sessions/update.go                    |  140 +++
 internal/tui/sessions/view.go                      |  118 ++
 main.go                                            |   85 +-
 pkg/assert/yaml.go                                 |  148 +++
 pkg/command/assertions.go                          |   96 ++
 pkg/fs/assertions.go                               |   62 +
 pkg/fs/grove.go                                    |   62 +-
 pkg/fs/temp.go                                     |   15 +
 pkg/harness/assertion.go                           |    8 +
 pkg/harness/ci.go                                  |    2 +-
 pkg/harness/context.go                             |  245 +++-
 pkg/harness/executor.go                            |   83 ++
 pkg/harness/harness.go                             |  460 +++++++-
 pkg/harness/mocks.go                               |   32 +
 pkg/harness/reporters/github.go                    |   23 +-
 pkg/harness/reporters/json.go                      |   14 +-
 pkg/harness/scenario.go                            |   67 ++
 pkg/harness/steps.go                               |   12 +-
 pkg/harness/ui.go                                  |  235 ++--
 pkg/project/config.go                              |  199 +++-
 pkg/recorder/recorder.go                           |  178 +++
 pkg/recorder/recorder_test.go                      |   57 +
 pkg/recorder/report.go                             |  312 +++++
 pkg/recorder/report.html.tpl                       |  111 ++
 pkg/recorder/session.go                            |   11 +
 pkg/tui/options.go                                 |    8 +
 pkg/tui/session.go                                 |  179 ++-
 pkg/tui/session_test.go                            |   82 ++
 pkg/ui/components.go                               |  131 ++-
 pkg/ui/renderer.go                                 |   29 +-
 pkg/ui/styles.go                                   |   97 --
 pkg/verify/collector.go                            |   75 ++
 pkg/verify/error.go                                |   27 +
 test-recorder.sh                                   |   48 +
 tests/e2e/fixtures/file-saver/go.mod               |   24 +
 tests/e2e/fixtures/file-saver/go.sum               |   37 +
 tests/e2e/fixtures/file-saver/main.go              |   75 ++
 tests/e2e/fixtures/list-tui/go.mod                 |   24 +
 tests/e2e/fixtures/list-tui/go.sum                 |   37 +
 tests/e2e/fixtures/list-tui/main.go                |  102 ++
 tests/e2e/fixtures/task-manager/go.mod             |   24 +
 tests/e2e/fixtures/task-manager/go.sum             |   37 +
 tests/e2e/fixtures/task-manager/main.go            |  189 ++++
 tests/e2e/main.go                                  |  110 ++
 tests/e2e/scenarios_assertions.go                  |  239 ++++
 tests/e2e/scenarios_cli.go                         |  231 ++++
 tests/e2e/scenarios_mocking.go                     |  157 +++
 tests/e2e/scenarios_parallel_runner.go             |  634 +++++++++++
 tests/e2e/scenarios_runner_tui.go                  |  375 ++++++
 tests/e2e/scenarios_sandboxing.go                  |   50 +
 tests/e2e/scenarios_setup_demo.go                  |  115 ++
 tests/e2e/scenarios_setup_teardown.go              |  563 +++++++++
 tests/e2e/scenarios_tui.go                         |  442 ++++++++
 .../e2e/tend/mocks/src}/docker/main.go             |    0
 .../e2e/tend/mocks/src}/flow/main.go               |    0
 .../mocks => tests/e2e/tend/mocks/src}/git/main.go |    0
 .../e2e/tend/mocks/src}/kubectl/main.go            |    0
 .../mocks => tests/e2e/tend/mocks/src}/llm/main.go |    0
 tests/e2e/tend/mocks/src/print-env/main.go         |   19 +
 tests/e2e/test_utils.go                            |   89 ++
 134 files changed, 12185 insertions(+), 5084 deletions(-)
```

## v0.4.1-nightly.9652d8b (2025-10-03)

## v0.4.0 (2025-10-01)

The test harness UI has been unified with the Grove ecosystem's core theme system (9202aca). This change replaces custom color palettes and table components with centralized, theme-aware elements, ensuring that the output of `tend` commands has a consistent look and feel with other tools in the ecosystem.

The documentation system has undergone a major overhaul to improve clarity and maintainability. The structure has been refactored into three distinct sections: Overview, Examples, and Conventions, with content updated to be more direct and align with an LLM-first design philosophy (4e2fcc3, fa57331). The documentation generator now supports automatic Table of Contents generation for the README (ae63f47, 59fc924) and a `strip_lines` setting for more concise generated content (a784516). Configuration and file naming conventions have also been standardized (3352846, 491e74b).

The CI workflow configuration has been updated to use `branches: [ none ]` to correctly disable automatic execution while maintaining valid syntax (fde50eb).

### Features

- Unify test harness UI with core theme system (9202aca)
- Add automated Table of Contents generation for documentation (ae63f47)
- Add `strip_lines` configuration to docgen for more succinct output (a784516)

### Bug Fixes

- Update CI workflow to use 'none' branches to correctly disable execution (fde50eb)

### Code Refactoring

- Standardize docgen.config.yml key order and settings (3352846)

### Documentation

- Refactor documentation structure into Overview, Examples, and Conventions (4e2fcc3)
- Update docgen configuration and README templates for TOC generation (59fc924)
- Rename Introduction sections to Overview for clarity (fa57331)
- Remove unused exclude_patterns from docgen config (df3a301)

### Chores

- Standardize documentation filenames to DD-name.md convention (491e74b)
- Temporarily disable CI workflow (63184b5)

### File Changes

```
 .github/workflows/ci.yml              |    4 +-
 README.md                             |  295 ++------
 docs/01-overview.md                   |   70 ++
 docs/02-examples.md                   |  252 +++++++
 docs/03-conventions.md                |  114 ++++
 docs/README.md.tpl                    |    5 +
 docs/best-practices.md                |   58 --
 docs/core-concepts.md                 |  108 ---
 docs/docgen.config.yml                |   51 +-
 docs/docs.rules                       |   27 +-
 docs/images/grove-tend-readme.svg     | 1192 +++++++++++++++++++++++++++++++++
 docs/introduction.md                  |   15 -
 docs/prompts/01-overview.md           |   58 ++
 docs/prompts/02-examples.md           |   39 ++
 docs/prompts/03-conventions.md        |   48 ++
 docs/prompts/best-practices.prompt.md |   27 -
 docs/prompts/core-concepts.prompt.md  |   29 -
 docs/prompts/introduction.prompt.md   |   12 -
 docs/prompts/usage-patterns.prompt.md |   29 -
 docs/usage-patterns.md                |  211 ------
 internal/cmd/list.go                  |   31 +-
 pkg/docs/docs.json                    |  106 ++-
 pkg/ui/styles.go                      |  111 ++-
 23 files changed, 1980 insertions(+), 912 deletions(-)
```

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

