package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Mock flow - usage: flow <command>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version":
		fmt.Println("Mock flow version 0.0.1-mock")
	case "list":
		fmt.Println("Mock flow - no workflows defined")
	default:
		fmt.Printf("Mock flow - unknown command: %s\n", os.Args[1])
	}

	fmt.Fprintf(os.Stderr, "[MOCK FLOW] Command: %s\n", os.Args[1])
}
