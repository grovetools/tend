package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mattsolo1/grove-tend/internal/harness"
	"github.com/mattsolo1/grove-tend/pkg/ui"
)

// newListCmd creates the list command with the provided scenarios
func newListCmd(allScenarios []*harness.Scenario) *cobra.Command {
	listCmd := &cobra.Command{
	Use:   "list",
	Short: "List available test scenarios",
	Long: `List all available test scenarios with their descriptions and tags.

This command helps you discover what scenarios are available and understand
their purpose before running them.

Examples:
  tend list                    # List all scenarios
  tend list --tags=smoke       # List scenarios tagged with 'smoke'
  tend list --verbose          # List with detailed information`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return listScenarios(cmd, args, allScenarios)
	},
	}
	
	return listCmd
}

func listScenarios(cmd *cobra.Command, args []string, allScenarios []*harness.Scenario) error {
	// Create UI renderer
	renderer := ui.NewRenderer(cmd.OutOrStdout(), verbose, 80)
	
	
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