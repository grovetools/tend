// File: tests/e2e/scenarios_runner_tui.go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mattsolo1/grove-tend/pkg/fs"
	"github.com/mattsolo1/grove-tend/pkg/git"
	"github.com/mattsolo1/grove-tend/pkg/harness"
	"github.com/mattsolo1/grove-tend/pkg/tui"
)

// TendTUIScenario tests the interactive `tend tui` command.
// It sets up mock projects with scenario definitions and verifies the TUI
// can display, navigate, filter, and focus on them correctly.
func TendTUIScenario() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"tend-tui-navigation",
		"Verifies the core interactive features of the `tend tui` command.",
		[]string{"tui", "interactive", "smoke"},
		[]harness.Step{
			harness.NewStep("Setup mock filesystem with projects", setupMockFilesystem),
			harness.NewStep("Launch TUI and verify initial state", launchTUIAndVerify),
			harness.NewStep("Test navigation and folding", testNavigationAndFolding),
			harness.NewStep("Test filtering (search)", testFiltering),
			harness.NewStep("Test focusing", testFocusing),
			harness.NewStep("Test help view", testHelpView),
			harness.NewStep("Quit the TUI", quitTUI),
		},
		true,  // localOnly = true, as it requires tmux
		false, // explicitOnly = false
	)
}

// setupMockFilesystem creates a sandboxed environment with mock projects for testing.
func setupMockFilesystem(ctx *harness.Context) error {
	// The harness provides RootDir which is the sandboxed working directory.
	// We need to set up a grove config that points to our test projects.

	// Create the grove config directory structure
	groveConfigDir := filepath.Join(ctx.ConfigDir(), "grove")
	if err := os.MkdirAll(groveConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create grove config dir: %w", err)
	}

	// Create a "code" directory in the sandboxed root to hold our test projects
	codeDir := filepath.Join(ctx.RootDir, "code")
	if err := os.MkdirAll(codeDir, 0755); err != nil {
		return fmt.Errorf("failed to create code dir: %w", err)
	}

	// Write global grove config that defines where to find groves
	// The groves config expects a map with named entries, not a list
	globalGroveConfig := fmt.Sprintf(`groves:
  test:
    path: %s
    enabled: true
`, codeDir)
	if err := fs.WriteString(filepath.Join(groveConfigDir, "grove.yml"), globalGroveConfig); err != nil {
		return fmt.Errorf("failed to write global grove config: %w", err)
	}

	// --- Project A (Standalone) ---
	projectADir := filepath.Join(codeDir, "project-a")
	projectAE2EDir := filepath.Join(projectADir, "tests", "e2e")
	if err := os.MkdirAll(projectAE2EDir, 0755); err != nil {
		return fmt.Errorf("failed to create project-a e2e dir: %w", err)
	}
	if err := fs.WriteString(filepath.Join(projectADir, "grove.yml"), "name: project-a\ntype: go\n"); err != nil {
		return err
	}
	if err := git.Init(projectADir); err != nil {
		return fmt.Errorf("failed to init git for project-a: %w", err)
	}
	if err := git.SetupTestConfig(projectADir); err != nil {
		return fmt.Errorf("failed to setup git config for project-a: %w", err)
	}

	// Add a dummy scenario file for discovery
	scenarioA := `package main

import "github.com/mattsolo1/grove-tend/pkg/harness"

func ScenarioForA() *harness.Scenario {
	return harness.NewScenario("standalone-test", "A standalone test scenario", []string{"smoke"}, nil)
}
`
	if err := fs.WriteString(filepath.Join(projectAE2EDir, "scenarios.go"), scenarioA); err != nil {
		return err
	}
	if err := git.Add(projectADir, "."); err != nil {
		return err
	}
	if err := git.Commit(projectADir, "init project-a"); err != nil {
		return err
	}

	// --- Project B (Ecosystem with Sub-project C) ---
	projectBDir := filepath.Join(codeDir, "project-b")
	subProjectCDir := filepath.Join(projectBDir, "sub-project-c")
	subProjectCE2EDir := filepath.Join(subProjectCDir, "tests", "e2e")
	if err := os.MkdirAll(subProjectCE2EDir, 0755); err != nil {
		return fmt.Errorf("failed to create sub-project-c e2e dir: %w", err)
	}

	// Project B is an ecosystem containing sub-project-c
	if err := fs.WriteString(filepath.Join(projectBDir, "grove.yml"), "name: project-b\nworkspaces:\n  - sub-project-c\n"); err != nil {
		return err
	}
	if err := git.Init(projectBDir); err != nil {
		return fmt.Errorf("failed to init git for project-b: %w", err)
	}
	if err := git.SetupTestConfig(projectBDir); err != nil {
		return fmt.Errorf("failed to setup git config for project-b: %w", err)
	}

	// Sub-project C
	if err := fs.WriteString(filepath.Join(subProjectCDir, "grove.yml"), "name: sub-project-c\ntype: go\n"); err != nil {
		return err
	}
	scenarioC := `package main

import "github.com/mattsolo1/grove-tend/pkg/harness"

func ScenarioForC() *harness.Scenario {
	return harness.NewScenario("sub-app-test", "A sub-project test scenario", []string{"integration"}, nil)
}
`
	if err := fs.WriteString(filepath.Join(subProjectCE2EDir, "scenarios.go"), scenarioC); err != nil {
		return err
	}
	if err := git.Add(projectBDir, "."); err != nil {
		return err
	}
	if err := git.Commit(projectBDir, "init project-b and sub-project-c"); err != nil {
		return err
	}

	return nil
}

// launchTUIAndVerify starts the tend tui and verifies initial state.
func launchTUIAndVerify(ctx *harness.Context) error {
	// Find the tend binary - use the binary from os.Args[0] which is the test runner,
	// and then find the tend binary in the same bin directory
	tendBinary, err := findTendBinary()
	if err != nil {
		return fmt.Errorf("failed to find tend binary: %w", err)
	}

	// StartTUI automatically injects sandboxed environment variables (HOME, XDG_CONFIG_HOME, etc.)
	// so the TUI will use our mock projects instead of discovering the real system workspaces
	session, err := ctx.StartTUI(tendBinary, []string{"tui"})
	if err != nil {
		return fmt.Errorf("failed to start `tend tui`: %w", err)
	}
	ctx.Set("tui_session", session)

	// Wait for the TUI to load
	if err := session.WaitForText("Tend Test Runner", 10*time.Second); err != nil {
		content, _ := session.Capture()
		return fmt.Errorf("TUI did not load: %w\nContent:\n%s", err, content)
	}

	// Verify both top-level projects are visible
	if err := session.AssertContains("project-a"); err != nil {
		content, _ := session.Capture()
		return fmt.Errorf("project-a not found: %w\nContent:\n%s", err, content)
	}
	if err := session.AssertContains("project-b"); err != nil {
		content, _ := session.Capture()
		return fmt.Errorf("project-b not found: %w\nContent:\n%s", err, content)
	}

	// Verify a discovered scenario is visible
	if err := session.AssertContains("standalone-test"); err != nil {
		content, _ := session.Capture()
		return fmt.Errorf("standalone-test scenario not found: %w\nContent:\n%s", err, content)
	}

	return nil
}

// testNavigationAndFolding tests basic UI navigation and expand/collapse.
func testNavigationAndFolding(ctx *harness.Context) error {
	session := ctx.Get("tui_session").(*tui.Session)

	// Test basic navigation using j/k keys
	// Navigate down with 'j' (vim-style down)
	if err := session.Type("j"); err != nil {
		return fmt.Errorf("failed to navigate down: %w", err)
	}

	// Navigate down again
	if err := session.Type("j"); err != nil {
		return fmt.Errorf("failed to navigate down again: %w", err)
	}

	// Test 'h' to collapse the current node
	// First navigate to project-a (has scenarios under it)
	// Use 'gg' to go to top first
	if err := session.Type("g", "g"); err != nil {
		return err
	}

	// Now navigate down to find project-a (it should be after project-b in the list based on the output)
	// Navigate to find scenarios.go file under project-a
	for i := 0; i < 10; i++ {
		if err := session.Type("j"); err != nil {
			return err
		}
	}

	// Test that both scenarios are visible
	if err := session.AssertContains("sub-app-test"); err != nil {
		content, _ := session.Capture()
		return fmt.Errorf("sub-app-test should be visible: %w\nContent:\n%s", err, content)
	}
	if err := session.AssertContains("standalone-test"); err != nil {
		content, _ := session.Capture()
		return fmt.Errorf("standalone-test should be visible: %w\nContent:\n%s", err, content)
	}

	return nil
}

// testFiltering tests the search/filter functionality.
func testFiltering(ctx *harness.Context) error {
	session := ctx.Get("tui_session").(*tui.Session)

	// Activate search with '/'
	if err := session.Type("/"); err != nil {
		return fmt.Errorf("failed to activate search: %w", err)
	}

	// Type search term
	if err := session.Type("sub-app-test"); err != nil {
		return fmt.Errorf("failed to type search term: %w", err)
	}

	// Verify the filtering is working - sub-app-test should be visible
	if err := session.WaitForText("sub-app-test", 3*time.Second); err != nil {
		content, _ := session.Capture()
		return fmt.Errorf("sub-app-test should be visible after search: %w\nContent:\n%s", err, content)
	}

	// Verify standalone-test is filtered out (not visible)
	if err := session.AssertNotContains("standalone-test"); err != nil {
		content, _ := session.Capture()
		return fmt.Errorf("standalone-test should be filtered out: %w\nContent:\n%s", err, content)
	}

	// Clear the filter by selecting all text and deleting it
	// First, send Escape to exit filter mode (blur the input)
	if err := session.Type("Escape"); err != nil {
		return fmt.Errorf("failed to send escape: %w", err)
	}

	// Re-enter search mode and clear the text
	if err := session.Type("/"); err != nil {
		return err
	}

	// Send Ctrl+U to clear the input line (common bash/readline shortcut)
	if err := session.Type("C-u"); err != nil {
		return err
	}
	if err := session.Type("Escape"); err != nil {
		return err
	}

	// Verify standalone-test reappeared
	if err := session.WaitForText("standalone-test", 3*time.Second); err != nil {
		content, _ := session.Capture()
		return fmt.Errorf("standalone-test should reappear after clearing filter: %w\nContent:\n%s", err, content)
	}

	return nil
}

// testFocusing tests the focus functionality.
func testFocusing(ctx *harness.Context) error {
	session := ctx.Get("tui_session").(*tui.Session)

	// First, make sure we're in a clean state - go to top of the list
	// Send 'gg' together as a single vim command
	if err := session.Type("g", "g"); err != nil {
		return err
	}

	// Navigate down to project-b using j keys
	// Based on the tree structure, project-b should be first
	// Navigate to project-b (should be near the top)
	if err := session.WaitForText("project-b", 2*time.Second); err != nil {
		content, _ := session.Capture()
		return fmt.Errorf("project-b not found: %w\nContent:\n%s", err, content)
	}

	// Use vim-style navigation to get to project-b
	for i := 0; i < 4; i++ {
		if err := session.Type("j"); err != nil {
			return err
		}
	}

	// Focus on the selected item with '.'
	if err := session.Type("."); err != nil {
		return fmt.Errorf("failed to send focus key: %w", err)
	}

	// Verify focus header appears (the header shows "Focus: <name>")
	if err := session.WaitForText("Focus:", 3*time.Second); err != nil {
		content, _ := session.Capture()
		return fmt.Errorf("Focus indicator not found: %w\nContent:\n%s", err, content)
	}

	// Clear focus with ctrl+g
	if err := session.Type("C-g"); err != nil {
		return fmt.Errorf("failed to clear focus: %w", err)
	}

	// Verify project-a is visible again (focus cleared)
	if err := session.WaitForText("project-a", 3*time.Second); err != nil {
		content, _ := session.Capture()
		return fmt.Errorf("project-a should be visible after clearing focus: %w\nContent:\n%s", err, content)
	}

	return nil
}

// testHelpView tests opening and closing the help dialog.
func testHelpView(ctx *harness.Context) error {
	session := ctx.Get("tui_session").(*tui.Session)

	// Open help with '?'
	if err := session.Type("?"); err != nil {
		return fmt.Errorf("failed to open help: %w", err)
	}

	if err := session.WaitForText("Tend Test Runner - Help", 3*time.Second); err != nil {
		content, _ := session.Capture()
		return fmt.Errorf("help view did not open: %w\nContent:\n%s", err, content)
	}

	// Close help with '?' again
	if err := session.Type("?"); err != nil {
		return fmt.Errorf("failed to close help: %w", err)
	}

	// Verify main view is back (check for project-a)
	if err := session.WaitForText("project-a", 3*time.Second); err != nil {
		content, _ := session.Capture()
		return fmt.Errorf("main view not restored after closing help: %w\nContent:\n%s", err, content)
	}

	return nil
}

// quitTUI exits the TUI cleanly.
func quitTUI(ctx *harness.Context) error {
	session := ctx.Get("tui_session").(*tui.Session)
	return session.Type("q")
}

// findTendBinary finds the tend binary path.
// It looks in the same bin directory as the current test runner binary.
func findTendBinary() (string, error) {
	// Get the path of the current executable (tend-e2e)
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	// The tend binary should be in the same directory as tend-e2e
	binDir := filepath.Dir(execPath)
	tendPath := filepath.Join(binDir, "tend")

	if _, err := os.Stat(tendPath); err != nil {
		// Try looking relative to the working directory
		wd, _ := os.Getwd()
		// Walk up to find the project root
		currentDir := wd
		for {
			groveYml := filepath.Join(currentDir, "grove.yml")
			if _, err := os.Stat(groveYml); err == nil {
				tendPath = filepath.Join(currentDir, "bin", "tend")
				if _, err := os.Stat(tendPath); err == nil {
					return tendPath, nil
				}
				break
			}
			parent := filepath.Dir(currentDir)
			if parent == currentDir {
				break
			}
			currentDir = parent
		}
		return "", fmt.Errorf("tend binary not found at %s or in project bin directory", tendPath)
	}

	return tendPath, nil
}
