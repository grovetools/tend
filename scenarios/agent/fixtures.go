package agent

import (
	"github.com/grovepm/grove-tend/pkg/fs"
)

// BasicGroveConfig returns a minimal grove.yml for testing
func BasicGroveConfig() *fs.GroveConfig {
	return &fs.GroveConfig{
		WorkspaceRoot: ".",
		Services: map[string]fs.ServiceSpec{
			"test-service": {
				Image:   "alpine:latest",
				Port:    8080,
				Command: []string{"sleep", "infinity"},
			},
		},
		Agent: &fs.AgentConfig{
			Enabled: true,
			Port:    8080,
			Image:   "alpine:latest",
		},
	}
}

// MultiServiceConfig returns a grove.yml with multiple services
func MultiServiceConfig() *fs.GroveConfig {
	return &fs.GroveConfig{
		WorkspaceRoot: ".",
		Services: map[string]fs.ServiceSpec{
			"web": {
				Image: "nginx:alpine",
				Port:  80,
			},
			"api": {
				Image:   "node:alpine",
				Port:    3000,
				Command: []string{"node", "-e", "require('http').createServer((req,res)=>res.end('OK')).listen(3000)"},
			},
			"db": {
				Image: "postgres:alpine",
				Port:  5432,
				Env: map[string]string{
					"POSTGRES_PASSWORD": "testpass",
				},
			},
		},
	}
}

// TestFiles returns common test files
func TestFiles() map[string]string {
	return map[string]string{
		"README.md": `# Test Workspace

This is a test workspace for Grove Tend tests.
`,
		".gitignore": `.grove/
*.tmp
`,
		"src/main.go": `package main

import "fmt"

func main() {
    fmt.Println("Hello from Grove test!")
}
`,
	}
}