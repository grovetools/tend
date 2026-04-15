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
