package agent

import (
	"context"
	"testing"

	"github.com/grovepm/grove-tend/internal/harness"
)

func TestAgentIsolationScenario(t *testing.T) {
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create harness with test options
	h := harness.New(harness.Options{
		Verbose:   true,
		NoCleanup: false,
	})

	// Run the scenario
	ctx := context.Background()
	result, err := h.Run(ctx, AgentIsolationScenario)

	if err != nil {
		t.Fatalf("Scenario failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("Scenario did not complete successfully")
	}

	t.Logf("Scenario completed in %v", result.Duration)
}

// TestAgentHelpers tests the helper functions
func TestAgentHelpers(t *testing.T) {
	t.Run("BasicGroveConfig", func(t *testing.T) {
		config := BasicGroveConfig()
		if len(config.Services) != 1 {
			t.Errorf("Expected 1 service, got %d", len(config.Services))
		}
	})

	t.Run("TestFiles", func(t *testing.T) {
		files := TestFiles()
		if _, ok := files["README.md"]; !ok {
			t.Error("Expected README.md in test files")
		}
	})
}