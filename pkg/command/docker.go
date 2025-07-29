package command

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Docker provides helpers for docker commands
type Docker struct{}

// NewDocker creates a new Docker command helper
func NewDocker() *Docker {
	return &Docker{}
}

// ContainerInfo represents basic container information
type ContainerInfo struct {
	ID      string
	Name    string
	Image   string
	Status  string
	Ports   []string
}

// ListContainers lists running containers
func (d *Docker) ListContainers(filter string) ([]ContainerInfo, error) {
	args := []string{"ps", "--format", "json"}
	if filter != "" {
		args = append(args, "--filter", filter)
	}

	output, err := RunSimple("docker", args...)
	if err != nil {
		return nil, err
	}

	var containers []ContainerInfo
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}

		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			return nil, fmt.Errorf("parsing container info: %w", err)
		}

		container := ContainerInfo{
			ID:     getString(raw, "ID"),
			Name:   getString(raw, "Names"),
			Image:  getString(raw, "Image"),
			Status: getString(raw, "Status"),
		}

		if ports, ok := raw["Ports"].(string); ok && ports != "" {
			container.Ports = strings.Split(ports, ", ")
		}

		containers = append(containers, container)
	}

	return containers, nil
}

// ContainerExists checks if a container exists
func (d *Docker) ContainerExists(name string) (bool, error) {
	containers, err := d.ListContainers(fmt.Sprintf("name=%s", name))
	if err != nil {
		return false, err
	}
	return len(containers) > 0, nil
}

// StopContainer stops a container
func (d *Docker) StopContainer(name string) error {
	_, err := RunSimple("docker", "stop", name)
	return err
}

// RemoveContainer removes a container
func (d *Docker) RemoveContainer(name string) error {
	_, err := RunSimple("docker", "rm", "-f", name)
	return err
}

// Logs gets container logs
func (d *Docker) Logs(container string, tail int) (string, error) {
	args := []string{"logs", container}
	if tail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", tail))
	}

	return RunSimple("docker", args...)
}

// getString safely extracts a string from a map
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}