package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"

	"github.com/mattsolo1/grove-tend/pkg/harness"
	"github.com/mattsolo1/grove-tend/pkg/ui"
)

// newListCmd creates the list command with the provided scenarios
func newListCmd(allScenarios []*harness.Scenario) *cobra.Command {
	var keyword string
	
	listCmd := &cobra.Command{
	Use:   "list",
	Short: "List available test scenarios",
	Long: `List all available test scenarios with their descriptions and tags.

This command helps you discover what scenarios are available and understand
their purpose before running them.

Examples:
  tend list                         # List all scenarios
  tend list --tags=smoke            # List scenarios tagged with 'smoke'
  tend list --keyword=git           # List scenarios containing 'git' in name, description, or tags
  tend list --verbose               # List with detailed information`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return listScenarios(cmd, args, allScenarios, keyword)
	},
	}
	
	listCmd.Flags().StringVarP(&keyword, "keyword", "k", "", "Filter scenarios by keyword (searches name, description, and tags)")
	
	return listCmd
}

func listScenarios(cmd *cobra.Command, args []string, allScenarios []*harness.Scenario, keyword string) error {
	// Create UI renderer
	renderer := ui.NewRenderer(cmd.OutOrStdout(), verbose, 80)
	
	
	// Filter scenarios by tags and keyword if specified
	filteredScenarios := filterScenarios(allScenarios, []string{}, tags)
	
	// Apply keyword filtering if specified
	if keyword != "" {
		filteredScenarios = filterByKeyword(filteredScenarios, keyword)
	}
	
	if len(filteredScenarios) == 0 {
		renderer.RenderInfo("No scenarios found matching the specified criteria")
		return nil
	}
	
	// Display header
	fmt.Printf("Available scenarios (%d):\n\n", len(filteredScenarios))
	
	// Build table data
	headers := []string{"NAME", "DESCRIPTION", "LOCAL", "EXPLICIT", "TAGS", "STEPS"}
	var rows [][]string
	
	for _, scenario := range filteredScenarios {
		// Format tags
		tagStr := "-"
		if len(scenario.Tags) > 0 {
			tagStr = strings.Join(scenario.Tags, ", ")
		}
		
		// Format description - truncate if too long
		description := scenario.Description
		if len(description) > 50 && !verbose {
			description = description[:47] + "..."
		}
		
		// Format local and explicit indicators
		localIndicator := ""
		if scenario.LocalOnly {
			localIndicator = "✅"
		}
		
		explicitIndicator := ""
		if scenario.ExplicitOnly {
			explicitIndicator = "✅"
		}
		
		row := []string{
			scenario.Name,
			description,
			localIndicator,
			explicitIndicator,
			tagStr,
			fmt.Sprintf("%d", len(scenario.Steps)),
		}
		rows = append(rows, row)
	}
	
	// Create table renderer
	re := lipgloss.NewRenderer(os.Stdout)
	
	// Define styles
	baseStyle := re.NewStyle().Padding(0, 1)
	headerStyle := baseStyle.Copy().Bold(true).Foreground(lipgloss.Color("#5FAFFF"))
	
	// Create the table
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086"))).
		Headers(headers...).
		Rows(rows...)
	
	// Apply styling
	t.StyleFunc(func(row, col int) lipgloss.Style {
		if row == 0 {
			return headerStyle
		}
		// Style based on column
		switch col {
		case 0: // Name column
			return baseStyle.Copy().Foreground(lipgloss.Color("#00D4AA")).Bold(true)
		case 2, 3: // Local and Explicit columns (centered)
			return baseStyle.Copy().Align(lipgloss.Center)
		case 4: // Tags column
			return baseStyle.Copy().Foreground(lipgloss.Color("#5FAFFF"))
		case 5: // Steps column
			return baseStyle.Copy().Foreground(lipgloss.Color("#00D787"))
		default:
			return baseStyle
		}
	})
	
	fmt.Println(t)
	
	// If verbose, show detailed step information for each scenario
	if verbose {
		fmt.Println("\nDetailed scenario information:")
		for _, scenario := range filteredScenarios {
			displayScenarioDetails(renderer, scenario)
		}
	}
	
	return nil
}

func displayScenarioDetails(renderer *ui.Renderer, scenario *harness.Scenario) {
	// Scenario name and description
	fmt.Printf("\n%s %s\n", 
		ui.HeaderStyle.Render("●"), 
		ui.TitleStyle.Render(scenario.Name))
	
	if scenario.Description != "" {
		fmt.Printf("  %s\n", ui.MutedStyle.Render(scenario.Description))
	}
	
	// Show LocalOnly warning
	if scenario.LocalOnly {
		fmt.Printf("  %s This scenario is marked as local-only and will be skipped in CI environments\n",
			ui.WarningStyle.Render("⚠"))
	}
	
	// Show ExplicitOnly warning
	if scenario.ExplicitOnly {
		fmt.Printf("  %s This scenario must be run explicitly by name (skipped during 'tend run')\n",
			ui.WarningStyle.Render("⚠"))
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

// filterByKeyword filters scenarios based on a keyword search in name, description, and tags
func filterByKeyword(scenarios []*harness.Scenario, keyword string) []*harness.Scenario {
	var filtered []*harness.Scenario
	lowercaseKeyword := strings.ToLower(keyword)
	
	for _, scenario := range scenarios {
		// Check if keyword appears in scenario name
		if strings.Contains(strings.ToLower(scenario.Name), lowercaseKeyword) {
			filtered = append(filtered, scenario)
			continue
		}
		
		// Check if keyword appears in description
		if strings.Contains(strings.ToLower(scenario.Description), lowercaseKeyword) {
			filtered = append(filtered, scenario)
			continue
		}
		
		// Check if keyword appears in any tag
		for _, tag := range scenario.Tags {
			if strings.Contains(strings.ToLower(tag), lowercaseKeyword) {
				filtered = append(filtered, scenario)
				break
			}
		}
	}
	
	return filtered
}