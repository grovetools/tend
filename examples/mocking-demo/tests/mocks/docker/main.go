package main

import (
	"fmt"
	"os"
	"strings"
)

// Mock docker command that simulates common docker operations
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: docker [OPTIONS] COMMAND")
		fmt.Println()
		fmt.Println("A self-sufficient runtime for containers (mock)")
		os.Exit(1)
	}

	command := os.Args[1]
	
	switch command {
	case "version":
		fmt.Println("Docker version 24.0.0-mock, build abcdef0")
		fmt.Println("Mock Docker for testing")
	case "ps":
		// Check for -a flag
		showAll := false
		for _, arg := range os.Args[2:] {
			if arg == "-a" || arg == "--all" {
				showAll = true
				break
			}
		}
		
		fmt.Println("CONTAINER ID   IMAGE          COMMAND                  CREATED         STATUS         PORTS     NAMES")
		if showAll {
			fmt.Println("abc123def456   nginx:latest   \"nginx -g 'daemon of…\"   2 hours ago     Exited (0)     80/tcp    webserver")
		}
		fmt.Println("def456ghi789   redis:alpine   \"docker-entrypoint.s…\"   5 minutes ago   Up 5 minutes   6379/tcp  cache")
		
	case "images":
		fmt.Println("REPOSITORY   TAG       IMAGE ID       CREATED       SIZE")
		fmt.Println("nginx        latest    abc123def456   2 weeks ago   142MB")
		fmt.Println("redis        alpine    def456ghi789   3 weeks ago   32.3MB")
		
	case "pull":
		if len(os.Args) < 3 {
			fmt.Println("Error: image name required")
			os.Exit(1)
		}
		image := os.Args[2]
		fmt.Printf("Using default tag: latest\n")
		fmt.Printf("latest: Pulling from library/%s\n", strings.Split(image, ":")[0])
		fmt.Printf("Digest: sha256:1234567890abcdef\n")
		fmt.Printf("Status: Downloaded newer image for %s\n", image)
		
	case "run":
		// Simple mock - just print that container started
		fmt.Println("container-id-123456")
		
	case "stop":
		if len(os.Args) < 3 {
			fmt.Println("Error: container name or ID required")
			os.Exit(1)
		}
		fmt.Println(os.Args[2])
		
	case "rm":
		if len(os.Args) < 3 {
			fmt.Println("Error: container name or ID required")
			os.Exit(1)
		}
		fmt.Println(os.Args[2])
		
	default:
		fmt.Fprintf(os.Stderr, "docker: '%s' is not a docker command (mock)\n", command)
		os.Exit(1)
	}
	
	// Log the command for debugging
	fmt.Fprintf(os.Stderr, "[MOCK DOCKER] Executed: docker %s\n", strings.Join(os.Args[1:], " "))
}