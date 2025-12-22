// File: tests/e2e/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mattsolo1/grove-tend/pkg/app"
	"github.com/mattsolo1/grove-tend/pkg/harness"
)

func main() {
	// A list of all E2E scenarios for tend itself.
	scenarios := []*harness.Scenario{
		// CLI Scenarios
		// TODO: TestKeywordFilteringScenario is disabled because it relies on discovering
		// scenarios from source files, but e2e scenarios are compiled into the binary.
		// Either create actual scenario files or test the filtering logic directly.
		// TestKeywordFilteringScenario(),
		LocalOnlyScenario(),
		ExplicitOnlyScenario(),

		// TUI Scenarios
		// AutoPathMocksScenario(), // Commented out due to shell-specific tmux issues
		EnvPassingTestScenario(),
		TendTUIScenario(),
		ExampleAdvancedTuiNavigation(),
		ExampleConditionalFlowsAndRecording(),
		ExampleFilesystemInteractionScenario(),

		// Mocking Scenarios
		GitWorkflowScenario,
		DockerScenario,
		LLMIntegrationScenario,
		FlowMockScenario,
		MixedDependenciesScenario,
		EnvironmentSandboxingScenario(),

		// Assertion Scenarios
		HardAssertionPassScenario(),
		HardAssertionFailScenario(),
		SoftAssertionPassScenario(),
		SoftAssertionFailScenario(),
		MixedAssertionsScenario(),
		AssertionDescriptionsScenario(),
		MultipleVerifyBlocksScenario(),
		VerifyWithNilValuesScenario(),

		// Setup/Teardown Demo Scenarios
		SetupDemoScenario(),
		SetupDemoWithTeardownScenario(),

		// Setup/Teardown E2E Test Scenarios
		SetupPhaseBasicScenario(),
		SetupPhaseMultipleStepsScenario(),
		SetupPhaseFailureHandlingScenario(),
		SetupPhaseFailureScenario(), // explicit-only: intentionally fails
		TeardownPhaseBasicScenario(),
		TeardownPhaseMultipleStepsScenario(),
		TeardownPhaseAfterFailureScenario(), // explicit-only: intentionally fails
		TeardownPhaseFailureScenario(),
		FullLifecycleScenario(),
		SetupPhaseAssertionsScenario(),
		TeardownWithNoCleanupScenario(),
		SetupOnlyModeScenario(),
		ReusableSetupStepsScenario(),
	}

	// Setup signal handling for graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down...")
		cancel()
	}()

	// Execute the custom tend application with our scenarios.
	if err := app.Execute(ctx, scenarios); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
