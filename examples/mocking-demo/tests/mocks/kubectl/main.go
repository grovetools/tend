package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Mock kubectl - usage: kubectl <command>")
		os.Exit(1)
	}
	
	switch os.Args[1] {
	case "version":
		fmt.Println("Mock kubectl version: v1.0.0-mock")
	case "get":
		if len(os.Args) > 2 && os.Args[2] == "pods" {
			fmt.Println("No resources found in default namespace.")
		} else {
			fmt.Println("Mock kubectl - get command")
		}
	default:
		fmt.Printf("Mock kubectl - command: %s\n", os.Args[1])
	}
}