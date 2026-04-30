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
				tuimuxBin, err := exec.LookPath("tuimux")
				if err != nil {
					return fmt.Errorf("tuimux binary not found in PATH: %w", err)
				}

				socketDir := ctx.NewDir("tuimux-socket")
				if err := os.MkdirAll(socketDir, 0o755); err != nil {
					return err
				}
				socketPath := filepath.Join(socketDir, "test.sock")
				ctx.Set("socket_path", socketPath)
				ctx.Set("tuimux_bin", tuimuxBin)

				// Start detached session (starts daemon + creates session + runs headless)
				cmd := exec.Command(tuimuxBin, "new", "-s", "test", "--socket", socketPath, "-d", "--wait")
				cmd.Stdout = os.Stderr
				cmd.Stderr = os.Stderr
				if err := cmd.Start(); err != nil {
					return fmt.Errorf("start tuimux: %w", err)
				}
				ctx.Set("tuimux_pid", cmd.Process.Pid)

				// Wait for the command to complete (--wait blocks until shell is idle)
				errCh := make(chan error, 1)
				go func() { errCh <- cmd.Wait() }()

				select {
				case err := <-errCh:
					if err != nil {
						return fmt.Errorf("tuimux new failed: %w", err)
					}
				case <-time.After(30 * time.Second):
					_ = cmd.Process.Kill()
					return fmt.Errorf("tuimux new timed out")
				}

				return nil
			}),

			harness.NewStep("Connect tend session and assert initial state", func(ctx *harness.Context) error {
				socketPath := ctx.GetString("socket_path")
				session := tui.NewTuimuxSession(socketPath, "test")
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

				// Find a panel to send keys to
				snap, err := session.GetDebugState()
				if err != nil {
					return err
				}

				var targetID string
				for id := range snap.Panels {
					targetID = id
					break
				}

				panel := session.Panel(targetID)
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
				socketPath := ctx.GetString("socket_path")
				_ = os.Remove(socketPath)
				return nil
			}),
		},
		true,  // localOnly
		false, // explicitOnly
	)
}
