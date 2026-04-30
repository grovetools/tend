package tui

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// newDebugHTTPClient creates an http.Client that connects via a Unix domain socket.
func newDebugHTTPClient(socketPath string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
		Timeout: 5 * time.Second,
	}
}

// tuimuxTransport wraps an http.RoundTripper and rewrites /debug/* paths
// to /api/debug/*?session=<name> so tend's existing locator code works
// transparently against a tuimux daemon.
type tuimuxTransport struct {
	base        http.RoundTripper
	sessionName string
}

func (t *tuimuxTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(req.URL.Path) >= 6 && req.URL.Path[:6] == "/debug" {
		req = req.Clone(req.Context())
		req.URL.Path = "/api" + req.URL.Path
		q := req.URL.Query()
		q.Set("session", t.sessionName)
		req.URL.RawQuery = q.Encode()
	}
	return t.base.RoundTrip(req)
}

// newTuimuxDebugClient creates an http.Client that connects via Unix socket
// to a tuimux daemon and rewrites debug paths for session targeting.
func newTuimuxDebugClient(socketPath, sessionName string) *http.Client {
	base := &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		},
	}
	return &http.Client{
		Transport: &tuimuxTransport{base: base, sessionName: sessionName},
		Timeout:   5 * time.Second,
	}
}

// waitForDebugServer polls GET /debug/state until the server responds with 200 OK.
func waitForDebugServer(client *http.Client, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	pollInterval := 50 * time.Millisecond

	for time.Now().Before(deadline) {
		resp, err := client.Get("http://unix/debug/state")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(pollInterval)
	}

	return fmt.Errorf("debug server not ready after %v", timeout)
}
