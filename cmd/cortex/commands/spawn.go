package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kareemaly/cortex1/internal/cli/sdk"
	"github.com/spf13/cobra"
)

var spawnJSONFlag bool

var spawnCmd = &cobra.Command{
	Use:   "spawn <ticket-id>",
	Short: "Spawn a session for a ticket",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ticketID := args[0]
		client := sdk.DefaultClient()

		// First find the ticket to get its status
		ticket, err := client.FindTicketByID(ticketID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Spawn a session
		session, err := client.SpawnSession(ticket.Status, ticket.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if spawnJSONFlag {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(session); err != nil {
				fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
				os.Exit(1)
			}
			return
		}

		fmt.Printf("Session spawned: %s\n", session.ID)
		fmt.Printf("  Ticket: %s (%s)\n", ticket.Title, ticket.ID)
		fmt.Printf("  Agent:  %s\n", session.Agent)
		fmt.Printf("  Tmux:   %s\n", session.TmuxWindow)
	},
}

func init() {
	spawnCmd.Flags().BoolVar(&spawnJSONFlag, "json", false, "Output as JSON")
	rootCmd.AddCommand(spawnCmd)
}
