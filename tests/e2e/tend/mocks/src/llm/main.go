package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Mock llm command that simulates an LLM CLI tool
func main() {
	// Simple state management via environment variable
	callCount := os.Getenv("LLM_MOCK_CALL_COUNT")
	if callCount == "" {
		callCount = "0"
	}

	// Parse command line
	var prompt string
	var jsonOutput bool

	for i, arg := range os.Args[1:] {
		if arg == "--json" {
			jsonOutput = true
		} else if arg == "-p" || arg == "--prompt" {
			if i+1 < len(os.Args)-1 {
				prompt = os.Args[i+2]
			}
		} else if !strings.HasPrefix(arg, "-") && prompt == "" {
			// Assume non-flag arguments are the prompt
			prompt = strings.Join(os.Args[i+1:], " ")
			break
		}
	}

	if prompt == "" {
		fmt.Fprintf(os.Stderr, "Error: No prompt provided\n")
		fmt.Fprintf(os.Stderr, "Usage: llm [--json] [-p|--prompt] <prompt>\n")
		os.Exit(1)
	}

	// Generate mock response based on prompt
	var response string
	if strings.Contains(strings.ToLower(prompt), "test") {
		response = "This is a mock response for testing purposes. The LLM mock received your prompt about testing."
	} else if strings.Contains(strings.ToLower(prompt), "hello") {
		response = "Hello! I'm a mock LLM. How can I help you today?"
	} else if strings.Contains(strings.ToLower(prompt), "code") {
		response = "Here's a simple example:\n```python\ndef mock_function():\n    return 'This is mock code'\n```"
	} else {
		response = fmt.Sprintf("Mock LLM response to: '%s'. This is call number %s to the mock.", prompt, callCount)
	}

	// Output response
	if jsonOutput {
		output := map[string]interface{}{
			"prompt":   prompt,
			"response": response,
			"model":    "mock-llm-v1",
			"tokens":   len(strings.Fields(prompt)) + len(strings.Fields(response)),
		}
		jsonBytes, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Println(response)
	}

	// Log for debugging
	fmt.Fprintf(os.Stderr, "[MOCK LLM] Prompt: %s\n", prompt)
}
