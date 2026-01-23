package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/spf13/cobra"
)

var ticketShowJSONFlag bool

var ticketShowCmd = &cobra.Command{
	Use:   "<id>",
	Short: "Show ticket details",
	Long:  `Show detailed information about a ticket, including its session and comments.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectPath, err := resolveProjectPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		ticketID := args[0]
		client := sdk.DefaultClient(projectPath)

		ticket, err := client.FindTicketByID(ticketID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if ticketShowJSONFlag {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(ticket); err != nil {
				fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// Print ticket info
		fmt.Printf("Ticket: %s\n", ticket.ID)
		fmt.Printf("  Title:  %s\n", ticket.Title)
		fmt.Printf("  Status: %s\n", ticket.Status)
		fmt.Printf("  Created: %s\n", ticket.Dates.Created.Format("Jan 02 15:04:05"))
		fmt.Printf("  Updated: %s\n", ticket.Dates.Updated.Format("Jan 02 15:04:05"))
		fmt.Println()

		// Print body
		if ticket.Body != "" {
			fmt.Println("Body:")
			fmt.Println(ticket.Body)
			fmt.Println()
		}

		// Print session info if present
		if ticket.Session != nil {
			session := ticket.Session
			fmt.Printf("Session: %s\n", session.ID)
			fmt.Printf("  Agent:       %s\n", session.Agent)
			fmt.Printf("  Started:     %s\n", session.StartedAt.Format("Jan 02 15:04:05"))
			if session.EndedAt != nil {
				fmt.Printf("  Ended:       %s\n", session.EndedAt.Format("Jan 02 15:04:05"))
			} else {
				fmt.Printf("  Ended:       (active)\n")
			}
			fmt.Printf("  Tmux Window: %s\n", session.TmuxWindow)
			fmt.Println()

			// Print current status
			if session.CurrentStatus != nil {
				fmt.Println("Current Status:")
				fmt.Printf("  Status: %s\n", session.CurrentStatus.Status)
				if session.CurrentStatus.Tool != nil {
					fmt.Printf("  Tool:   %s\n", *session.CurrentStatus.Tool)
				}
				if session.CurrentStatus.Work != nil {
					fmt.Printf("  Work:   %s\n", *session.CurrentStatus.Work)
				}
				fmt.Println()
			}
		}

		// Print comments
		if len(ticket.Comments) > 0 {
			fmt.Println("Comments:")
			for _, c := range ticket.Comments {
				fmt.Printf("  [%s] %s\n", c.Type, c.Content)
			}
		}
	},
}

func init() {
	ticketShowCmd.Flags().BoolVar(&ticketShowJSONFlag, "json", false, "Output as JSON")
}
