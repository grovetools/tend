package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/grovepm/grove-tend/internal/harness"
	"github.com/grovepm/grove-tend/pkg/command"
	"github.com/grovepm/grove-tend/pkg/fs"
)

// WaitForAgent waits for the agent to be ready
func WaitForAgent(ctx *harness.Context, dir string, timeout time.Duration) error {
	grove := command.NewGrove(ctx.GroveBinary).InDir(dir)

	deadline := time.Now().Add(timeout)
	lastError := ""
	for time.Now().Before(deadline) {
		result := grove.Run("agent", "status")
		if result.ExitCode == 0 {
			// Agent command succeeded, check if it's running
			output := strings.ToLower(result.Stdout)
			if strings.Contains(output, "running") || strings.Contains(output, "up") {
				return nil
			}
			lastError = fmt.Sprintf("agent status: %s", result.Stdout)
		} else {
			lastError = fmt.Sprintf("agent status failed (exit %d): %s", result.ExitCode, result.Stderr)
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("agent did not become ready within %v (last status: %s)", timeout, lastError)
}

// GetAgentContainerName extracts the agent container name from status
func GetAgentContainerName(ctx *harness.Context, dir string) (string, error) {
	grove := command.NewGrove(ctx.GroveBinary).InDir(dir)

	// Try to get status output
	status, err := grove.Output("agent", "status")
	if err != nil {
		return "", fmt.Errorf("failed to get agent status: %w", err)
	}

	// Look for the Name: field in the output
	lines := strings.Split(status, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Name:") {
			// Extract the container name after "Name:"
			parts := strings.SplitN(line, ":", 2)
			if len(parts) >= 2 {
				name := strings.TrimSpace(parts[1])
				if name != "" {
					return name, nil
				}
			}
		}
	}

	// Fallback: look for any line containing "grove-" and "-agent"
	for _, line := range lines {
		if strings.Contains(line, "grove-") && strings.Contains(line, "-agent") {
			// Try to extract the container name
			if idx := strings.Index(line, "grove-"); idx >= 0 {
				name := line[idx:]
				// Find the end of the container name (space or end of line)
				if endIdx := strings.IndexAny(name, " \t\n"); endIdx > 0 {
					name = name[:endIdx]
				}
				return strings.TrimSpace(name), nil
			}
		}
	}

	return "", fmt.Errorf("could not find agent container name in status output:\n%s", status)
}

// VerifyAgentIsolation verifies that two agents are running in isolation
func VerifyAgentIsolation(ctx *harness.Context, container1, container2 string) error {
	docker := command.NewDocker()

	// Check both containers exist
	exists1, err := docker.ContainerExists(container1)
	if err != nil {
		return fmt.Errorf("checking container %s: %w", container1, err)
	}
	if !exists1 {
		return fmt.Errorf("container %s does not exist", container1)
	}

	exists2, err := docker.ContainerExists(container2)
	if err != nil {
		return fmt.Errorf("checking container %s: %w", container2, err)
	}
	if !exists2 {
		return fmt.Errorf("container %s does not exist", container2)
	}

	// Verify they have different names
	if container1 == container2 {
		return fmt.Errorf("agents are using the same container: %s", container1)
	}

	// Could add more isolation checks here (network, volumes, etc.)

	return nil
}

// CreateTestService creates a simple test service configuration
func CreateTestService(dir string, name string, port int) error {
	config := &fs.GroveConfig{
		WorkspaceRoot: ".",
		Services: map[string]fs.ServiceSpec{
			name: {
				Image:   "nginx:alpine",
				Port:    port,
				Command: []string{"nginx", "-g", "daemon off;"},
			},
		},
		Agent: &fs.AgentConfig{
			Enabled: true,
			Port:    8080,
		},
	}

	return fs.WriteGroveConfig(dir, config)
}