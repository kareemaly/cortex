package commands

import (
	"github.com/spf13/cobra"
)

var ticketCmd = &cobra.Command{
	Use:   "ticket",
	Short: "Manage tickets",
	Long:  `Commands for managing tickets: view ticket details.`,
}

func init() {
	ticketCmd.AddCommand(ticketShowCmd)
	rootCmd.AddCommand(ticketCmd)
}
