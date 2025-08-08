package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mattsolo1/grove-tend/internal/harness"
	"github.com/mattsolo1/grove-tend/pkg/app"
)

func main() {
	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down...")
		cancel()
	}()

	// No built-in scenarios - grove-tend is now a pure library
	// Scenarios should be defined in the repositories they test
	var allScenarios []*harness.Scenario

	// Execute the application
	if err := app.Execute(ctx, allScenarios); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}