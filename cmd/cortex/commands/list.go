package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listStatus string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tickets",
	Run: func(cmd *cobra.Command, args []string) {
		if listStatus != "" {
			fmt.Printf("not implemented yet (status: %s)\n", listStatus)
		} else {
			fmt.Println("not implemented yet")
		}
	},
}

func init() {
	listCmd.Flags().StringVar(&listStatus, "status", "", "Filter tickets by status")
	rootCmd.AddCommand(listCmd)
}
