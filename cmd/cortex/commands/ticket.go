package commands

import (
	"github.com/spf13/cobra"
)

var ticketCmd = &cobra.Command{
	Use:   "ticket",
	Short: "Manage tickets",
	Long:  `Commands for managing tickets: list, spawn sessions, and view ticket details.`,
}

func init() {
	ticketCmd.AddCommand(ticketListCmd)
	ticketCmd.AddCommand(ticketSpawnCmd)
	ticketCmd.AddCommand(ticketShowCmd)
	rootCmd.AddCommand(ticketCmd)
}
