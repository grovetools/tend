package recorder

import (
	"bytes"
	"io"
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
	cmd := exec.Command(command[0], command[1:]...)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}
	defer ptmx.Close()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			pty.InheritSize(os.Stdin, ptmx)
		}
	}()
	ch <- syscall.SIGWINCH
	defer signal.Stop(ch)

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	var frames []Frame
	startTime := time.Now()

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

	for {
		var currentInput bytes.Buffer
		var currentOutput bytes.Buffer

		input, ok := <-inputChan
		if !ok {
			break
		}

		currentInput.Write(input)
		ptmx.Write(input)
		os.Stdout.Write(input)

		for {
			var output bytes.Buffer
			mw := io.MultiWriter(&output, os.Stdout)

			buffer := make([]byte, 4096)
			ptmx.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
			n, err := ptmx.Read(buffer)
			if n > 0 {
				mw.Write(buffer[:n])
				currentOutput.Write(buffer[:n])
			}
			if err != nil {
				if os.IsTimeout(err) {
					break
				}
				goto end_loop
			}
		}

		frames = append(frames, Frame{
			Timestamp: time.Since(startTime),
			Input:     string(currentInput.Bytes()),
			Output:    string(currentOutput.Bytes()),
		})
	}

end_loop:

	// Wait for the command to finish, but ignore errors
	// as the user exiting is expected.
	_ = cmd.Wait()

	return frames, nil
}
