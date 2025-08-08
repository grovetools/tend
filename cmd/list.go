package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mattsolo1/grove-tend/internal/harness"
	"github.com/mattsolo1/grove-tend/pkg/ui"
	"github.com/mattsolo1/grove-tend/scenarios"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available test scenarios",
	Long: `List all available test scenarios with their descriptions and tags.

This command helps you discover what scenarios are available and understand
their purpose before running them.

Examples:
  tend list                    # List all scenarios
  tend list --tags=smoke       # List scenarios tagged with 'smoke'
  tend list --verbose          # List with detailed information`,
	RunE: listScenarios,
}

func listScenarios(cmd *cobra.Command, args []string) error {
	// Create UI renderer
	renderer := ui.NewRenderer(cmd.OutOrStdout(), verbose, 80)
	
	// Load scenarios
	scenarioLoader := scenarios.NewLoader(filepath.Join(rootDir, scenarioDir))
	allScenarios, err := scenarioLoader.LoadAll()
	if err != nil {
		renderer.RenderError(fmt.Errorf("failed to load scenarios: %w", err))
		return err
	}
	
	// Filter scenarios by tags if specified
	filteredScenarios := filterScenarios(allScenarios, []string{}, tags)
	
	if len(filteredScenarios) == 0 {
		renderer.RenderInfo("No scenarios found matching the specified criteria")
		return nil
	}
	
	// Display scenarios
	renderer.RenderList(fmt.Sprintf("Available scenarios (%d):", len(filteredScenarios)), []string{})
	
	for _, scenario := range filteredScenarios {
		displayScenario(renderer, scenario)
	}
	
	return nil
}

func displayScenario(renderer *ui.Renderer, scenario *harness.Scenario) {
	// Scenario name and description
	fmt.Printf("\n%s %s\n", 
		ui.HeaderStyle.Render("●"), 
		ui.TitleStyle.Render(scenario.Name))
	
	if scenario.Description != "" {
		fmt.Printf("  %s\n", ui.MutedStyle.Render(scenario.Description))
	}
	
	// Tags
	if len(scenario.Tags) > 0 {
		tagStr := strings.Join(scenario.Tags, ", ")
		fmt.Printf("  %s %s\n", 
			ui.InfoStyle.Render("Tags:"), 
			ui.MutedStyle.Render(tagStr))
	}
	
	// Step count
	fmt.Printf("  %s %d step(s)\n", 
		ui.InfoStyle.Render("Steps:"), 
		len(scenario.Steps))
	
	// If verbose, show step details
	if verbose {
		for i, step := range scenario.Steps {
			fmt.Printf("    %s %s\n", 
				ui.StepNumberStyle.Render(fmt.Sprintf("%d.", i+1)), 
				ui.StepNameStyle.Render(step.Name))
			
			if step.Description != "" {
				fmt.Printf("      %s\n", ui.MutedStyle.Render(step.Description))
			}
		}
	}
}