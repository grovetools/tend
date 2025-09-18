package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

// newDocsCmd creates the docs command with subcommands for generating and editing documentation
func newDocsCmd() *cobra.Command {
	docsCmd := &cobra.Command{
		Use:   "docs",
		Short: "Manage grove-tend documentation",
		Long:  `Commands for generating and managing the grove-tend documentation that is embedded in the binary.`,
	}

	docsCmd.AddCommand(newDocsGenerateCmd())
	docsCmd.AddCommand(newDocsRulesCmd())

	return docsCmd
}

// newDocsGenerateCmd creates the generate subcommand for docs
func newDocsGenerateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "generate",
		Short: "Generate tend-examples.xml and TEND_GUIDE.md",
		Long: `Generate the grove-tend documentation files from source examples.

This command must be run from within the grove-tend source directory.
It will:
1. Build context from docs/examples.cx.rules
2. Use an LLM to generate structured XML documentation
3. Transform the XML into human-readable Markdown
4. Save both files for embedding and distribution`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if we're in the grove-tend source directory
			if !isInTendSourceDir() {
				return fmt.Errorf("this command must be run from the grove-tend source directory")
			}

			// Check if the generate-docs tool exists
			toolPath := filepath.Join("cmd", "generate-docs", "main.go")
			if _, err := os.Stat(toolPath); os.IsNotExist(err) {
				return fmt.Errorf("generate-docs tool not found at %s", toolPath)
			}

			fmt.Println("Generating tend documentation...")
			
			// Run the generate-docs tool
			generateCmd := exec.Command("go", "run", toolPath)
			generateCmd.Stdout = os.Stdout
			generateCmd.Stderr = os.Stderr
			
			if err := generateCmd.Run(); err != nil {
				return fmt.Errorf("documentation generation failed: %w", err)
			}

			return nil
		},
	}
}

// newDocsRulesCmd creates the rules subcommand for docs
func newDocsRulesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rules",
		Short: "Open the documentation rules file in your editor",
		Long: `Open docs/examples.cx.rules in your configured editor.

This command must be run from within the grove-tend source directory.
The editor used is determined by the EDITOR environment variable.
If EDITOR is not set, it will try common editors (vim, vi, nano).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if we're in the grove-tend source directory
			if !isInTendSourceDir() {
				return fmt.Errorf("this command must be run from the grove-tend source directory")
			}

			rulesPath := filepath.Join("docs", "generation", "examples.cx.rules")
			
			// Check if the rules file exists
			if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
				return fmt.Errorf("rules file not found at %s", rulesPath)
			}

			// Get the editor from environment variable
			editor := os.Getenv("EDITOR")
			if editor == "" {
				// Try common editors
				for _, e := range []string{"vim", "vi", "nano"} {
					if _, err := exec.LookPath(e); err == nil {
						editor = e
						break
					}
				}
				if editor == "" {
					return fmt.Errorf("no editor found; please set the EDITOR environment variable")
				}
			}

			// Open the file in the editor
			editorCmd := exec.Command(editor, rulesPath)
			editorCmd.Stdin = os.Stdin
			editorCmd.Stdout = os.Stdout
			editorCmd.Stderr = os.Stderr
			
			if err := editorCmd.Run(); err != nil {
				return fmt.Errorf("failed to open editor: %w", err)
			}

			return nil
		},
	}
}

// isInTendSourceDir checks if the current directory is the grove-tend source directory
func isInTendSourceDir() bool {
	// Check for specific markers that indicate we're in the grove-tend source
	markers := []string{
		"grove.yml",           // Should have grove.yml
		"pkg/harness",         // Should have harness package
		"docs/generation/examples.cx.rules", // Should have the rules file
	}

	// Additionally check grove.yml content
	groveYml, err := os.ReadFile("grove.yml")
	if err != nil {
		return false
	}
	
	// Check if grove.yml indicates this is grove-tend
	if !contains(string(groveYml), "name: tend") {
		return false
	}

	// Check for existence of marker files/directories
	for _, marker := range markers {
		if _, err := os.Stat(marker); os.IsNotExist(err) {
			return false
		}
	}

	return true
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}