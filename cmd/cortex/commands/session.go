package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/spf13/cobra"
)

var sessionJSONFlag bool

var sessionCmd = &cobra.Command{
	Use:   "session <ticket-id>",
	Short: "Show session details for a ticket",
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

		if ticket.Session == nil {
			fmt.Fprintf(os.Stderr, "Error: ticket %s has no active session\n", ticket.ID)
			os.Exit(1)
		}
		session := ticket.Session

		if sessionJSONFlag {
			output := struct {
				Session *sdk.SessionResponse `json:"session"`
				Ticket  *sdk.TicketResponse  `json:"ticket"`
			}{
				Session: session,
				Ticket:  ticket,
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(output); err != nil {
				fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// Print session info
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

		// Print ticket info
		fmt.Printf("Ticket: %s\n", ticket.ID)
		fmt.Printf("  Title:  %s\n", ticket.Title)
		fmt.Printf("  Status: %s\n", ticket.Status)
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

		// Print comments for this session
		sessionComments := []sdk.CommentResponse{}
		for _, c := range ticket.Comments {
			if c.SessionID == session.ID {
				sessionComments = append(sessionComments, c)
			}
		}
		if len(sessionComments) > 0 {
			fmt.Println("Comments:")
			for _, c := range sessionComments {
				fmt.Printf("  [%s] %s\n", c.Type, c.Content)
			}
		}
	},
}

func init() {
	sessionCmd.Flags().BoolVar(&sessionJSONFlag, "json", false, "Output as JSON")
	rootCmd.AddCommand(sessionCmd)
}
