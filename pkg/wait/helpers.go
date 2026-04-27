package wait

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/grovetools/tend/pkg/command"
)

// ForFile waits for a file to exist
func ForFile(path string, timeout time.Duration) error {
	opts := DefaultOptions()
	opts.Timeout = timeout

	return For(func() (bool, error) {
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			return false, nil
		}
		return err == nil, err
	}, opts)
}

// ForFileContent waits for a file to contain specific content
func ForFileContent(path string, content string, timeout time.Duration) error {
	opts := DefaultOptions()
	opts.Timeout = timeout

	return ForWithMessage(func() (bool, string, error) {
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			return false, "file does not exist", nil
		}
		if err != nil {
			return false, "", err
		}

		fileContent := string(data)
		if strings.Contains(fileContent, content) {
			return true, "content found", nil
		}

		preview := fileContent
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}

		return false, fmt.Sprintf("content not found, file contains: %s", preview), nil
	}, opts)
}

// ForPort waits for a port to be open
func ForPort(host string, port int, timeout time.Duration) error {
	opts := DefaultOptions()
	opts.Timeout = timeout

	address := fmt.Sprintf("%s:%d", host, port)

	return ForWithMessage(func() (bool, string, error) {
		conn, err := net.DialTimeout("tcp", address, 2*time.Second)
		if err != nil {
			return false, fmt.Sprintf("port not reachable: %v", err), nil
		}
		conn.Close()
		return true, "port is open", nil
	}, opts)
}

// ForHTTP waits for an HTTP endpoint to be available
func ForHTTP(url string, expectedStatus int, timeout time.Duration) error {
	opts := DefaultOptions()
	opts.Timeout = timeout

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	return ForWithMessage(func() (bool, string, error) {
		resp, err := client.Get(url)
		if err != nil {
			return false, fmt.Sprintf("request failed: %v", err), nil
		}
		defer resp.Body.Close()

		if resp.StatusCode == expectedStatus {
			return true, fmt.Sprintf("got expected status %d", expectedStatus), nil
		}

		return false, fmt.Sprintf("got status %d, expected %d", resp.StatusCode, expectedStatus), nil
	}, opts)
}

// ForContainer waits for a container to exist
func ForContainer(containerName string, timeout time.Duration) error {
	opts := DefaultOptions()
	opts.Timeout = timeout

	docker := command.NewDocker()

	return For(func() (bool, error) {
		exists, err := docker.ContainerExists(containerName)
		return exists, err
	}, opts)
}

// ForContainerStatus waits for a container to reach a specific status
func ForContainerStatus(containerName string, expectedStatus string, timeout time.Duration) error {
	opts := DefaultOptions()
	opts.Timeout = timeout

	docker := command.NewDocker()

	return ForWithMessage(func() (bool, string, error) {
		containers, err := docker.ListContainers(fmt.Sprintf("name=%s", containerName))
		if err != nil {
			return false, "", err
		}

		if len(containers) == 0 {
			return false, "container not found", nil
		}

		status := containers[0].Status
		if strings.Contains(strings.ToLower(status), strings.ToLower(expectedStatus)) {
			return true, fmt.Sprintf("container is %s", status), nil
		}

		return false, fmt.Sprintf("container status is %s", status), nil
	}, opts)
}

// ForCommand waits for a command to succeed
func ForCommand(name string, args []string, timeout time.Duration) error {
	opts := DefaultOptions()
	opts.Timeout = timeout

	return For(func() (bool, error) {
		cmd := command.New(name, args...)
		result := cmd.Run()
		return result.ExitCode == 0, nil
	}, opts)
}

// ForOutput waits for a command to produce specific output
func ForOutput(name string, args []string, expected string, timeout time.Duration) error {
	opts := DefaultOptions()
	opts.Timeout = timeout

	return ForWithMessage(func() (bool, string, error) {
		cmd := command.New(name, args...)
		result := cmd.Run()

		if result.Error != nil && result.ExitCode != 0 {
			return false, fmt.Sprintf("command failed: %v", result.Error), nil
		}

		output := result.Stdout + result.Stderr
		if strings.Contains(output, expected) {
			return true, "expected output found", nil
		}

		preview := output
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}

		return false, fmt.Sprintf("output was: %s", preview), nil
	}, opts)
}
