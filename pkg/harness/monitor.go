package harness

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/grovetools/tend/pkg/command"
)

// ContainerMonitor monitors Docker containers during test execution
type ContainerMonitor struct {
	mu          sync.Mutex
	running     bool
	ctx         context.Context
	cancel      context.CancelFunc
	updateFunc  func(containers []ContainerInfo)
	filter      string
	interval    time.Duration
}

// ContainerInfo holds simplified container information
type ContainerInfo struct {
	Image   string
	Created string
	Names   string
}

// NewContainerMonitor creates a new container monitor
func NewContainerMonitor(filter string, interval time.Duration) *ContainerMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	return &ContainerMonitor{
		ctx:      ctx,
		cancel:   cancel,
		filter:   filter,
		interval: interval,
	}
}

// Start begins monitoring containers
func (m *ContainerMonitor) Start(updateFunc func(containers []ContainerInfo)) {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.updateFunc = updateFunc
	m.mu.Unlock()

	go m.monitor()
}

// Stop stops monitoring containers
func (m *ContainerMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.running {
		m.cancel()
		m.running = false
	}
}

// monitor runs the monitoring loop
func (m *ContainerMonitor) monitor() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	// Initial update
	m.updateContainers()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.updateContainers()
		}
	}
}

// updateContainers fetches current container list and calls update function
func (m *ContainerMonitor) updateContainers() {
	// Use custom format to get only the fields we want
	cmd := command.New("docker", "ps", "--format", "table {{.Image}}\t{{.CreatedAt}}\t{{.Names}}")
	if m.filter != "" {
		cmd = command.New("docker", "ps", "--filter", m.filter, "--format", "table {{.Image}}\t{{.CreatedAt}}\t{{.Names}}")
	}
	
	result := cmd.Run()
	if result.Error != nil {
		return
	}

	// Parse the output
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	if len(lines) <= 1 {
		// No containers or only header
		m.updateFunc([]ContainerInfo{})
		return
	}

	var containers []ContainerInfo
	for i, line := range lines {
		if i == 0 && strings.Contains(line, "IMAGE") {
			// Skip header
			continue
		}
		
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			// Handle multi-word fields by reconstructing
			image := fields[0]
			names := fields[len(fields)-1]
			
			// Created time is everything in between
			created := strings.Join(fields[1:len(fields)-1], " ")
			
			containers = append(containers, ContainerInfo{
				Image:   image,
				Created: created,
				Names:   names,
			})
		}
	}

	m.updateFunc(containers)
}

// GetContainerSnapshot returns a current snapshot of containers
func GetContainerSnapshot(filter string) ([]ContainerInfo, error) {
	cmd := command.New("docker", "ps", "--format", "table {{.Image}}\t{{.CreatedAt}}\t{{.Names}}")
	if filter != "" {
		cmd = command.New("docker", "ps", "--filter", filter, "--format", "table {{.Image}}\t{{.CreatedAt}}\t{{.Names}}")
	}
	
	result := cmd.Run()
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get containers: %w", result.Error)
	}

	// Parse the output
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	if len(lines) <= 1 {
		return []ContainerInfo{}, nil
	}

	var containers []ContainerInfo
	for i, line := range lines {
		if i == 0 && strings.Contains(line, "IMAGE") {
			continue
		}
		
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			image := fields[0]
			names := fields[len(fields)-1]
			created := strings.Join(fields[1:len(fields)-1], " ")
			
			containers = append(containers, ContainerInfo{
				Image:   image,
				Created: created,
				Names:   names,
			})
		}
	}

	return containers, nil
}