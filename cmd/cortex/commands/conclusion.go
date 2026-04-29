package commands

import "github.com/spf13/cobra"

var conclusionCmd = &cobra.Command{
	Use:   "conclusion",
	Short: "Inspect conclusions",
}

func init() {
	rootCmd.AddCommand(conclusionCmd)
}
