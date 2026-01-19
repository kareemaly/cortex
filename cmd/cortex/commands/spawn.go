package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var spawnCmd = &cobra.Command{
	Use:   "spawn <ticket-id>",
	Short: "Spawn a ticket session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("not implemented yet (ticket: %s)\n", args[0])
	},
}

func init() {
	rootCmd.AddCommand(spawnCmd)
}
