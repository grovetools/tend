package drive

import (
	"fmt"
	"time"

	"github.com/grovetools/tend/pkg/tui"
)

// Mode names the attach path used to reach the running app.
const (
	// ModeTuimux attaches to a named session on a tuimux daemon socket.
	ModeTuimux = "tuimux"
	// ModeDebugSocket attaches to an app's raw debug Unix socket (treemux-style
	// TREEMUX_DEBUG_SOCKET).
	ModeDebugSocket = "debug-socket"
)

// AttachOptions selects the attach path. When Session is set, the tuimux daemon
// path is used; otherwise Socket is treated as a raw app debug socket.
type AttachOptions struct {
	Socket       string
	Session      string
	ReadyTimeout time.Duration
}

// Mode reports which attach path AttachOptions selects.
func (o AttachOptions) Mode() string {
	if o.Session != "" {
		return ModeTuimux
	}
	return ModeDebugSocket
}

// Attach connects to an already-running app's debug socket and returns a Driver.
// It never spawns anything (attach-only v1).
//
//   - tuimux mode (Session set): tui.NewTuimuxSession(socket, session) followed
//     by WaitForTuimuxReady. The tuimux transport rewrites /debug/* endpoints to
//     /api/debug/*?session=<name>.
//   - debug-socket mode (Session empty): a bare Session with SetDebugSocket,
//     which dials the raw /debug/* endpoints and blocks until the server is
//     ready.
//
// In both modes the mux engine is intentionally left unset: the driver only
// uses the debug-socket plane.
func Attach(opts AttachOptions) (Driver, error) {
	if opts.Socket == "" {
		return nil, fmt.Errorf("socket path is required")
	}
	timeout := opts.ReadyTimeout
	if timeout <= 0 {
		timeout = defaultStepTimeout
	}

	if opts.Session != "" {
		s := tui.NewTuimuxSession(opts.Socket, opts.Session)
		if err := s.WaitForTuimuxReady(timeout); err != nil {
			return nil, fmt.Errorf("attach tuimux session %q on %s: %w", opts.Session, opts.Socket, err)
		}
		return SessionDriver{Session: s}, nil
	}

	s := tui.NewSession("", nil, "")
	if err := s.SetDebugSocket(opts.Socket, timeout); err != nil {
		return nil, fmt.Errorf("attach debug socket %s: %w", opts.Socket, err)
	}
	return SessionDriver{Session: s}, nil
}
