package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/kareemaly/cortex1/internal/cli/sdk"
	"github.com/spf13/cobra"
)

var sessionJSONFlag bool

var sessionCmd = &cobra.Command{
	Use:   "session <id>",
	Short: "Show session details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectPath, err := resolveProjectPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		sessionID := args[0]
		client := sdk.DefaultClient(projectPath)

		session, ticket, err := client.FindSession(sessionID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

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

		// Print report summary
		if session.Report.Summary != "" {
			fmt.Println("Report:")
			fmt.Printf("  Summary: %s\n", session.Report.Summary)
			if len(session.Report.Files) > 0 {
				fmt.Printf("  Files:   %s\n", strings.Join(session.Report.Files, ", "))
			}
			if len(session.Report.Decisions) > 0 {
				fmt.Println("  Decisions:")
				for _, d := range session.Report.Decisions {
					fmt.Printf("    - %s\n", d)
				}
			}
		}
	},
}

func init() {
	sessionCmd.Flags().BoolVar(&sessionJSONFlag, "json", false, "Output as JSON")
	rootCmd.AddCommand(sessionCmd)
}
