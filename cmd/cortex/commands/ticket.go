package commands

import "github.com/spf13/cobra"

var ticketCmd = &cobra.Command{
	Use:   "ticket",
	Short: "Inspect tickets",
}

func init() {
	rootCmd.AddCommand(ticketCmd)
}
