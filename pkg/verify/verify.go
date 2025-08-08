package verify

import (
	"fmt"
	"strings"
	"time"
	
	"github.com/mattsolo1/grove-tend/pkg/command"
	"github.com/mattsolo1/grove-tend/pkg/wait"
)

// ServiceRunning verifies a Grove service is running
func ServiceRunning(groveBinary, workDir, serviceName string) error {
	grove := command.NewGrove(groveBinary).InDir(workDir)
	
	// First wait for the service to appear in status
	err := wait.ForWithMessage(func() (bool, string, error) {
		output, err := grove.Output("status")
		if err != nil {
			return false, "grove status failed", err
		}
		
		if strings.Contains(output, serviceName) {
			return true, fmt.Sprintf("service %s found in status", serviceName), nil
		}
		
		return false, fmt.Sprintf("service %s not in status output", serviceName), nil
	}, wait.Options{
		Timeout:      30 * time.Second,
		PollInterval: 1 * time.Second,
	})
	
	if err != nil {
		return fmt.Errorf("service %s not running: %w", serviceName, err)
	}
	
	return nil
}

// ContainersRunning verifies multiple containers are running
func ContainersRunning(containerNames ...string) error {
	for _, name := range containerNames {
		// Wait for container to exist
		if err := wait.ForContainer(name, 30*time.Second); err != nil {
			return fmt.Errorf("container %s not found: %w", name, err)
		}
		
		// Wait for it to be running
		if err := wait.ForContainerStatus(name, "running", 30*time.Second); err != nil {
			return fmt.Errorf("container %s not running: %w", name, err)
		}
	}
	
	return nil
}

// PortsAccessible verifies multiple ports are accessible
func PortsAccessible(host string, ports ...int) error {
	for _, port := range ports {
		if err := wait.ForPort(host, port, 30*time.Second); err != nil {
			return fmt.Errorf("port %d not accessible: %w", port, err)
		}
	}
	return nil
}

// HTTPEndpoints verifies multiple HTTP endpoints
func HTTPEndpoints(endpoints map[string]int) error {
	for url, expectedStatus := range endpoints {
		if err := wait.ForHTTP(url, expectedStatus, 30*time.Second); err != nil {
			return fmt.Errorf("endpoint %s failed: %w", url, err)
		}
	}
	return nil
}

// NoErrors verifies no errors in logs
func NoErrors(containerName string, errorPatterns []string) error {
	docker := command.NewDocker()
	
	logs, err := docker.Logs(containerName, 100)
	if err != nil {
		return fmt.Errorf("getting logs: %w", err)
	}
	
	for _, pattern := range errorPatterns {
		if strings.Contains(logs, pattern) {
			// Extract context around the error
			lines := strings.Split(logs, "\n")
			for i, line := range lines {
				if strings.Contains(line, pattern) {
					start := i - 2
					if start < 0 {
						start = 0
					}
					end := i + 3
					if end > len(lines) {
						end = len(lines)
					}
					
					context := strings.Join(lines[start:end], "\n")
					return fmt.Errorf("found error pattern '%s' in logs:\n%s", pattern, context)
				}
			}
		}
	}
	
	return nil
}