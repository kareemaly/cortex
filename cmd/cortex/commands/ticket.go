package commands

import (
	"github.com/spf13/cobra"
)

var ticketCmd = &cobra.Command{
	Use:   "ticket [id]",
	Short: "Manage tickets",
	Long:  `Commands for managing tickets: view ticket details.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			// Delegate to the show subcommand
			ticketShowCmd.Run(cmd, args)
			return
		}
		_ = cmd.Help()
	},
}

func init() {
	ticketCmd.AddCommand(ticketShowCmd)
	rootCmd.AddCommand(ticketCmd)
}
