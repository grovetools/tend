package demo

import (
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/grovetools/core/pkg/tmux"
)

// Teardown stops every process the demo environment spawned: the demo tmux
// server, the demo-scoped tuimux daemon, and any groved daemons started by
// delegated CLI commands (grove/nb/flow run with GROVE_HOME inside the demo
// dir, so their PID files live under the demo tree).
//
// It must run BEFORE the demo directory is removed, while PID files and
// sockets are still readable. All steps are best-effort: already-dead
// processes and missing files are ignored.
func Teardown(ctx context.Context, demoDir string, meta *Metadata) {
	if meta != nil {
		// Kill the demo tmux server (tmux backend).
		if meta.TmuxSocket != "" {
			if client, err := tmux.NewClientWithSocket(meta.TmuxSocket); err == nil {
				_ = client.KillServer(ctx)
			}
		}

		// Kill the demo tuimux daemon (tuimux backend).
		killTuimuxDaemon(demoDir, meta)
	}

	// Kill any groved daemons that recorded PID files inside the demo tree.
	killGrovedDaemons(demoDir)
}

// killTuimuxDaemon terminates the tuimux daemon spawned for this demo.
// Prefers the PID recorded in metadata; for demos created before the PID was
// recorded, it falls back to matching the daemon by its socket path in the
// process table.
func killTuimuxDaemon(demoDir string, meta *Metadata) {
	if meta.TuimuxDaemonPID > 1 {
		signalPID(meta.TuimuxDaemonPID)
		return
	}

	// Legacy metadata: locate the daemon via its per-demo socket path, which
	// appears verbatim in the daemon's command line ("tuimux daemon --socket <path>").
	socketPath := meta.TuimuxSocket
	if socketPath == "" {
		socketPath = TuimuxSocketPath(demoDir)
	}
	if _, err := os.Stat(socketPath); err != nil {
		return // No socket — no daemon was started (tmux backend) or it already exited.
	}
	out, err := exec.Command("pgrep", "-f", socketPath).Output()
	if err != nil {
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if pid, err := strconv.Atoi(strings.TrimSpace(line)); err == nil {
			signalPID(pid)
		}
	}
}

// killGrovedDaemons finds groved PID files anywhere under root and kills the
// recorded PIDs. Matches both the unscoped "groved.pid" and scoped
// "groved-<name>-<hash>.pid" variants (groved writes them to its state dir,
// which is inside the demo tree because GROVE_HOME points at the demo).
func killGrovedDaemons(root string) {
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasPrefix(name, "groved") || !strings.HasSuffix(name, ".pid") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if pid, parseErr := strconv.Atoi(strings.TrimSpace(string(data))); parseErr == nil {
			signalPID(pid)
		}
		return nil
	})
}

// signalPID sends SIGTERM to a process, ignoring failures (already dead,
// permission, etc.). The daemons handle SIGTERM for graceful shutdown.
func signalPID(pid int) {
	if pid <= 1 {
		return
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = proc.Signal(syscall.SIGTERM)
}
