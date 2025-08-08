package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/mattsolo1/grove-core/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
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

func init() {
	versionCmd.Flags().Bool("json", false, "Output version information in JSON format")
	rootCmd.AddCommand(versionCmd)
}