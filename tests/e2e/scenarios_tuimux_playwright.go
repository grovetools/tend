package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/grovetools/tend/pkg/harness"
	"github.com/grovetools/tend/pkg/tui"
)

// TuimuxPlaywrightTestScenario exercises the tuimux debug endpoints via
// tend's PanelLocator API: state queries, focus changes, key injection,
// pane splits, and overlay toggling.
func TuimuxPlaywrightTestScenario() *harness.Scenario {
	return harness.NewScenarioWithOptions(
		"tuimux-playwright-test",
		"End-to-end Playwright-style testing of tuimux debug endpoints",
		[]string{"tuimux", "playwright", "debug", "slow"},
		[]harness.Step{
			harness.NewStep("Start tuimux daemon + detached session", func(ctx *harness.Context) error {
				tuimuxBin := os.Getenv("TUIMUX_BIN")
				if tuimuxBin == "" {
					home, _ := os.UserHomeDir()
					candidate := filepath.Join(home, "Code/grovetools/.grove-worktrees/mux-engine-extraction/tuimux/bin/tuimux")
					if _, serr := os.Stat(candidate); serr == nil {
						tuimuxBin = candidate
					}
				}
				if tuimuxBin == "" {
					var err error
					tuimuxBin, err = exec.LookPath("tuimux")
					if err != nil {
						return fmt.Errorf("tuimux not found (set TUIMUX_BIN): %w", err)
					}
				}

				ctx.Set("tuimux_bin", tuimuxBin)

				// Kill any existing test session
				_ = exec.Command(tuimuxBin, "kill-session", "-t", "tend-pw-test").Run()

				// Start detached session using default socket (daemon auto-starts)
				cmd := exec.Command(tuimuxBin, "new", "-s", "tend-pw-test", "-d")
				cmd.Stdout = os.Stderr
				cmd.Stderr = os.Stderr
				if err := cmd.Start(); err != nil {
					return fmt.Errorf("start tuimux: %w", err)
				}
				ctx.Set("tuimux_pid", cmd.Process.Pid)

				// Wait for the session to be ready (hub connection + shell spawn)
				time.Sleep(3 * time.Second)
				return nil
			}),

			harness.NewStep("Connect tend session and assert initial state", func(ctx *harness.Context) error {
				home, _ := os.UserHomeDir()
				socketPath := filepath.Join(home, ".local", "state", "tuimux", "daemon.sock")
				session := tui.NewTuimuxSession(socketPath, "tend-pw-test")
				ctx.Set("session", session)

				if err := session.WaitForTuimuxReady(10 * time.Second); err != nil {
					return fmt.Errorf("debug endpoint not ready: %w", err)
				}

				// Assert initial panel exists and is focused
				panel := session.Panel("shell-0")
				if err := panel.AssertExists(); err != nil {
					return fmt.Errorf("initial panel not found: %w", err)
				}
				if err := panel.AssertFocused(); err != nil {
					return fmt.Errorf("initial panel not focused: %w", err)
				}

				// Verify debug state has exactly 1 panel
				snap, err := session.GetDebugState()
				if err != nil {
					return fmt.Errorf("get debug state: %w", err)
				}
				if len(snap.Panels) != 1 {
					return fmt.Errorf("expected 1 panel, got %d", len(snap.Panels))
				}
				if snap.ActivePanelID != "shell-0" {
					return fmt.Errorf("expected active panel shell-0, got %s", snap.ActivePanelID)
				}

				return nil
			}),

			harness.NewStep("Split pane and assert 2 panels", func(ctx *harness.Context) error {
				session := ctx.Get("session").(*tui.Session)

				if err := session.ExecuteTuimuxCommand([]string{"split-window", "-h"}); err != nil {
					return fmt.Errorf("split-window: %w", err)
				}

				// Wait for state to settle, then check panel count
				time.Sleep(500 * time.Millisecond)

				snap, err := session.GetDebugState()
				if err != nil {
					return fmt.Errorf("get debug state after split: %w", err)
				}
				if len(snap.Panels) != 2 {
					return fmt.Errorf("expected 2 panels after split, got %d", len(snap.Panels))
				}

				return nil
			}),

			harness.NewStep("Send keys to panel and wait for text", func(ctx *harness.Context) error {
				session := ctx.Get("session").(*tui.Session)

				// Target shell-0 which has a running shell (pane-1 may still be spawning)
				panel := session.Panel("shell-0")
				if err := panel.SendKeys("echo PLAYWRIGHT_TEST_OK\n"); err != nil {
					return fmt.Errorf("send keys: %w", err)
				}

				if err := panel.WaitForText("PLAYWRIGHT_TEST_OK", 5*time.Second); err != nil {
					return fmt.Errorf("text not found after send keys: %w", err)
				}

				return nil
			}),

			harness.NewStep("Open session browser overlay and verify", func(ctx *harness.Context) error {
				session := ctx.Get("session").(*tui.Session)

				if err := session.ExecuteTuimuxCommand([]string{"run-command", "choose-tree"}); err != nil {
					return fmt.Errorf("run choose-tree: %w", err)
				}

				time.Sleep(200 * time.Millisecond)

				snap, err := session.GetDebugState()
				if err != nil {
					return fmt.Errorf("get debug state: %w", err)
				}
				if !snap.Overlays.TreeActive {
					return fmt.Errorf("expected tree overlay to be active")
				}

				return nil
			}),

			harness.NewStep("Dismiss overlay and verify normal state", func(ctx *harness.Context) error {
				session := ctx.Get("session").(*tui.Session)

				if err := session.ExecuteTuimuxCommand([]string{"run-command", "dismiss"}); err != nil {
					return fmt.Errorf("run dismiss: %w", err)
				}

				time.Sleep(200 * time.Millisecond)

				snap, err := session.GetDebugState()
				if err != nil {
					return fmt.Errorf("get debug state: %w", err)
				}
				if snap.Overlays.TreeActive {
					return fmt.Errorf("expected tree overlay to be dismissed")
				}
				if snap.Overlays.HelpActive {
					return fmt.Errorf("expected help overlay to be dismissed")
				}

				return nil
			}),

			harness.NewStep("Cleanup", func(ctx *harness.Context) error {
				tuimuxBin := ctx.GetString("tuimux_bin")
				_ = exec.Command(tuimuxBin, "kill-session", "-t", "tend-pw-test").Run()
				return nil
			}),
		},
		true,  // localOnly
		false, // explicitOnly
	)
}
