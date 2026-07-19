package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/grovetools/tend/pkg/drive"
)

// newDriveCmd creates the `drive` command: a scripted TUI driver that attaches
// to an already-running app's debug socket, replays a step list, and emits a
// deterministic evidence bundle.
func newDriveCmd() *cobra.Command {
	var (
		socket      string
		session     string
		outDir      string
		stepTimeout time.Duration
	)

	driveCmd := &cobra.Command{
		Use:   "drive --socket <path> [--session <name>] <script.yaml>",
		Short: "Replay a scripted TUI session against a running app's debug socket",
		Long: `Attach to an already-running app's debug socket (no spawn), replay an
ordered list of steps from a YAML script, and write a deterministic evidence
bundle for agent QA loops.

Attach modes:
  --socket <path> --session <name>   Attach to a named session on a tuimux
                                     daemon socket.
  --socket <path>                    Attach to an app's raw debug socket
                                     (treemux-style TREEMUX_DEBUG_SOCKET).

Script schema (a flat, ordered YAML list; one key per step):
  - type: "<keys>"                    Inject keystrokes into the active panel.
  - kittykey: {panel: "<id>", keycode: <int>, mods: <int>}
                                      Inject a synthesized CSI-u key event.
  - wait: {}                          Wait for the rendered state to stabilize.
  - wait: {timeout: 5s}               ...with a per-step timeout override.
  - assert_contains: "<text>"         Assert the rendered state contains text.
  - assert_pattern: "<regexp>"        Assert the rendered state matches a regexp.
  - assert_structural: {active_panel: "<id>", rail_active: "<id|label>",
                        focused: "<id>", focused_count: <n>,
                        panel_type: {"<id>": "<type>"}}
                                      Assert fields of the structural debug
                                      state; all fields optional, at least one
                                      required.
  - snapshot: "<label>"               Write <label>.txt + <label>.json evidence.

Exit codes:
  0   all steps passed
  1   infrastructure / parse / attach error
  2   an assertion failed but the run completed to the failure point

Example:
  tend drive --socket $SOCK --session my-session script.yaml --out ./drive-evidence`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scriptPath := args[0]

			data, err := os.ReadFile(scriptPath)
			if err != nil {
				return fmt.Errorf("read script %s: %w", scriptPath, err)
			}
			steps, err := drive.ParseScript(data)
			if err != nil {
				return fmt.Errorf("parse script %s: %w", scriptPath, err)
			}

			attachOpts := drive.AttachOptions{
				Socket:       socket,
				Session:      session,
				ReadyTimeout: stepTimeout,
			}
			driver, err := drive.Attach(attachOpts)
			if err != nil {
				return err
			}

			runner := &drive.Runner{
				Driver:         driver,
				Steps:          steps,
				OutDir:         outDir,
				DefaultTimeout: stepTimeout,
			}
			result := runner.Run()

			meta := drive.ManifestMeta{
				Socket:  socket,
				Session: session,
				Mode:    attachOpts.Mode(),
				Script:  scriptPath,
			}
			manifest := drive.BuildManifest(meta, result)
			if err := drive.WriteManifest(outDir, manifest); err != nil {
				return fmt.Errorf("write evidence bundle: %w", err)
			}

			exitCode := result.ExitCode()
			fmt.Fprintf(cmd.OutOrStdout(), "Evidence bundle written to %s (exit %d)\n", outDir, exitCode)

			if exitCode != 0 {
				reportFailure(result)
				os.Exit(exitCode)
			}
			return nil
		},
	}

	driveCmd.Flags().StringVar(&socket, "socket", "", "Path to the app debug socket or tuimux daemon socket (required)")
	driveCmd.Flags().StringVar(&session, "session", "", "tuimux session name (selects the tuimux attach path)")
	driveCmd.Flags().StringVar(&outDir, "out", "./drive-evidence", "Directory for the evidence bundle")
	driveCmd.Flags().DurationVar(&stepTimeout, "timeout", 10*time.Second, "Default per-step wait/ready timeout")
	_ = driveCmd.MarkFlagRequired("socket")

	return driveCmd
}

// reportFailure prints the failing step and its diagnostic snapshot to stderr,
// per the drive exit-code contract (the bundle is most valuable exactly then).
func reportFailure(result *drive.RunResult) {
	if result.FailedIndex == 0 {
		return
	}
	failed := result.Steps[result.FailedIndex-1]
	fmt.Fprintf(os.Stderr, "\n=== drive failed at step %d (%s: %s) ===\n",
		failed.Index, failed.Kind, failed.Arg)
	fmt.Fprintf(os.Stderr, "outcome: %s\n", failed.Outcome)
	if failed.Failure != "" {
		fmt.Fprintf(os.Stderr, "reason:  %s\n", failed.Failure)
	}
	if result.Diagnostic != "" {
		fmt.Fprintf(os.Stderr, "\n%s\n", result.Diagnostic)
	}
}
