package fs

import (
	"fmt"
	"path/filepath"
)

// CreateProjectStructure creates a basic project structure for testing
func CreateProjectStructure(root string) error {
	// Create common directories
	dirs := []string{
		"src",
		"tests",
		"docs",
		".git",
	}

	for _, dir := range dirs {
		if err := CreateDir(filepath.Join(root, dir)); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
	}

	// Create some test files
	files := map[string]string{
		"README.md": "# Test Project\n\nThis is a test project for tend testing.",
		"src/main.go": `package main

import "fmt"

func main() {
    fmt.Println("Hello, Grove!")
}
`,
		".gitignore": "*.tmp\n.grove/\n",
	}

	for path, content := range files {
		fullPath := filepath.Join(root, path)
		if err := WriteString(fullPath, content); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}

	return nil
}

// CreateServiceFiles creates files for a test service
func CreateServiceFiles(root, serviceName string) error {
	serviceDir := filepath.Join(root, "services", serviceName)
	if err := CreateDir(serviceDir); err != nil {
		return err
	}

	// Create Dockerfile
	dockerfile := fmt.Sprintf(`FROM alpine:latest
LABEL service="%s"
CMD ["sleep", "infinity"]
`, serviceName)

	if err := WriteString(filepath.Join(serviceDir, "Dockerfile"), dockerfile); err != nil {
		return fmt.Errorf("writing Dockerfile: %w", err)
	}

	return nil
}
