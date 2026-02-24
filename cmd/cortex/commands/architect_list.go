package commands

import (
	"fmt"

	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/spf13/cobra"
)

var architectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered architects",
	RunE: func(cmd *cobra.Command, args []string) error {
		ensureDaemon()

		client := sdk.DefaultClient("")
		resp, err := client.ListArchitects()
		if err != nil {
			return fmt.Errorf("failed to list architects: %w", err)
		}

		if len(resp.Architects) == 0 {
			fmt.Println("No architects registered. Use 'cortex architect create' to create one.")
			return nil
		}

		for _, a := range resp.Architects {
			status := ""
			if !a.Exists {
				status = " (missing)"
			}
			title := a.Title
			if title == "" {
				title = a.Path
			}
			fmt.Printf("  %s%s\n    %s\n", title, status, a.Path)
			if a.Counts != nil {
				fmt.Printf("    Tickets: %d backlog, %d in progress, %d done\n",
					a.Counts.Backlog, a.Counts.Progress, a.Counts.Done)
			}
		}

		return nil
	},
}

func init() {
	architectCmd.AddCommand(architectListCmd)
}
