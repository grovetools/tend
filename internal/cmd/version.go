package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/grovetools/core/version"
	"github.com/spf13/cobra"
)

// newVersionCmd creates the version command
func newVersionCmd() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version information for this binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOutput, _ := cmd.Flags().GetBool("json")
			info := version.GetInfo()

			if jsonOutput {
				jsonData, err := json.MarshalIndent(info, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal version info to JSON: %w", err)
				}
				fmt.Println(string(jsonData))
			} else {
				fmt.Println(info.String())
			}
			return nil
		},
	}

	versionCmd.Flags().Bool("json", false, "Output version information in JSON format")
	return versionCmd
}
