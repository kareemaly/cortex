package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var architectCmd = &cobra.Command{
	Use:   "architect",
	Short: "Start or attach to architect session",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("not implemented yet")
	},
}

func init() {
	rootCmd.AddCommand(architectCmd)
}
