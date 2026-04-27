package recorder

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// Recorder manages the recording of a terminal session.
type Recorder struct{}

// New creates a new Recorder.
func New() *Recorder {
	return &Recorder{}
}

// Run executes the command in a PTY and records the session.
func (r *Recorder) Run(command []string) ([]Frame, error) {
	cmd := exec.Command(command[0], command[1:]...) //nolint:gosec // command from caller

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}
	defer ptmx.Close()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			_ = pty.InheritSize(os.Stdin, ptmx)
		}
	}()
	ch <- syscall.SIGWINCH
	defer signal.Stop(ch)

	// Check if stdin is a terminal before making it raw
	stdinFd := int(os.Stdin.Fd())
	if !term.IsTerminal(stdinFd) {
		return nil, fmt.Errorf("stdin is not a terminal - recording requires an interactive terminal")
	}

	oldState, err := term.MakeRaw(stdinFd)
	if err != nil {
		return nil, fmt.Errorf("failed to set terminal to raw mode: %w", err)
	}
	defer func() { _ = term.Restore(stdinFd, oldState) }()

	var frames []Frame
	startTime := time.Now()

	// Channel for input from user
	inputChan := make(chan []byte)
	go func() {
		for {
			buf := make([]byte, 1024)
			n, err := os.Stdin.Read(buf)
			if err != nil {
				close(inputChan)
				return
			}
			inputChan <- buf[:n]
		}
	}()

	// Channel for output from PTY
	outputChan := make(chan []byte, 10)
	doneChan := make(chan struct{})
	go func() {
		defer close(doneChan)
		defer close(outputChan)
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				outputChan <- data
			}
			if err != nil {
				return
			}
		}
	}()

	// Main loop: handle input and output concurrently
	var currentInput bytes.Buffer
	var currentOutput bytes.Buffer
	outputStableTimer := time.NewTimer(0)
	<-outputStableTimer.C // Drain initial timer

	collectingOutput := false

	for {
		select {
		case input, ok := <-inputChan:
			if !ok {
				goto end_loop
			}

			// If we were collecting output for a previous input, save that frame first
			if collectingOutput && currentInput.Len() > 0 {
				frames = append(frames, Frame{
					Timestamp: time.Since(startTime),
					Input:     currentInput.String(),
					Output:    currentOutput.String(),
				})
				currentInput.Reset()
				currentOutput.Reset()
			}

			// Write input to PTY
			currentInput.Write(input)
			_, err := ptmx.Write(input)
			if err != nil {
				goto end_loop
			}

			// Start collecting output for this input
			collectingOutput = true
			outputStableTimer.Reset(100 * time.Millisecond)

		case output, ok := <-outputChan:
			if !ok {
				goto end_loop
			}
			// Write output to stdout AND capture it
			os.Stdout.Write(output)

			if collectingOutput {
				currentOutput.Write(output)
				// Reset stability timer - output is still coming
				outputStableTimer.Reset(100 * time.Millisecond)
			}

		case <-outputStableTimer.C:
			// Output has stabilized - save the frame
			if collectingOutput && currentInput.Len() > 0 {
				frames = append(frames, Frame{
					Timestamp: time.Since(startTime),
					Input:     currentInput.String(),
					Output:    currentOutput.String(),
				})
				currentInput.Reset()
				currentOutput.Reset()
				collectingOutput = false
			}

		case <-doneChan:
			// PTY closed (command exited)
			goto end_loop
		}
	}

end_loop:

	// Save any remaining frame
	if currentInput.Len() > 0 || currentOutput.Len() > 0 {
		frames = append(frames, Frame{
			Timestamp: time.Since(startTime),
			Input:     currentInput.String(),
			Output:    currentOutput.String(),
		})
	}

	// Wait for the command to finish
	_ = cmd.Wait()

	return frames, nil
}
