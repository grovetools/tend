package main

import (
	"fmt"
	"os"
	"strings"
)

// Mock git command that simulates common git operations
func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: git <command> [<args>]")
		os.Exit(1)
	}

	command := os.Args[1]
	
	switch command {
	case "init":
		fmt.Println("Initialized empty Git repository in .git/")
	case "status":
		fmt.Println("On branch main")
		fmt.Println("No commits yet")
		fmt.Println()
		fmt.Println("nothing to commit (create/copy files and use \"git add\" to track)")
	case "add":
		// Silent success
	case "commit":
		// Look for -m flag
		messageIndex := -1
		for i, arg := range os.Args {
			if arg == "-m" && i+1 < len(os.Args) {
				messageIndex = i + 1
				break
			}
		}
		if messageIndex > 0 {
			fmt.Printf("[main (root-commit) abc1234] %s\n", os.Args[messageIndex])
			fmt.Println(" 0 files changed")
			fmt.Println(" create mode 100644 test.txt")
		} else {
			fmt.Println("error: missing commit message")
			os.Exit(1)
		}
	case "log":
		fmt.Println("commit abc1234567890abcdef1234567890abcdef12 (HEAD -> main)")
		fmt.Println("Author: Test User <test@example.com>")
		fmt.Println("Date:   Mon Jan 1 00:00:00 2024 +0000")
		fmt.Println()
		fmt.Println("    Initial commit")
	case "remote":
		if len(os.Args) > 2 && os.Args[2] == "add" {
			fmt.Printf("remote '%s' added\n", os.Args[3])
		}
	case "push":
		fmt.Println("Everything up-to-date")
	default:
		fmt.Fprintf(os.Stderr, "git: '%s' is not a git command (mock)\n", command)
		os.Exit(1)
	}
	
	// Always log that the mock was called
	fmt.Fprintf(os.Stderr, "[MOCK GIT] Executed: git %s\n", strings.Join(os.Args[1:], " "))
}