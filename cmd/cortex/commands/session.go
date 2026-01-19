package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var sessionCmd = &cobra.Command{
	Use:   "session <id>",
	Short: "Open session view TUI",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("not implemented yet (id: %s)\n", args[0])
	},
}

func init() {
	rootCmd.AddCommand(sessionCmd)
}
