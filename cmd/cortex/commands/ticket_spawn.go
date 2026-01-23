package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/spf13/cobra"
)

var (
	ticketSpawnJSONFlag   bool
	ticketSpawnResumeFlag bool
	ticketSpawnFreshFlag  bool
)

var ticketSpawnCmd = &cobra.Command{
	Use:   "spawn <id>",
	Short: "Spawn a session for a ticket",
	Long: `Spawn an agent session for a ticket.

The agent will be assigned to work on the specified ticket and can interact
with it through MCP tools.

Flags:
  --resume    Resume an orphaned session (if one exists)
  --fresh     Clear existing session state and start fresh`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Validate mutually exclusive flags
		if ticketSpawnResumeFlag && ticketSpawnFreshFlag {
			fmt.Fprintf(os.Stderr, "Error: --resume and --fresh are mutually exclusive\n")
			os.Exit(1)
		}

		projectPath, err := resolveProjectPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		ticketID := args[0]
		client := sdk.DefaultClient(projectPath)

		// First find the ticket to get its status
		ticket, err := client.FindTicketByID(ticketID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Determine mode from flags
		mode := ""
		if ticketSpawnResumeFlag {
			mode = "resume"
		} else if ticketSpawnFreshFlag {
			mode = "fresh"
		}

		// Spawn a session
		session, err := client.SpawnSession(ticket.Status, ticket.ID, mode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if ticketSpawnJSONFlag {
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
	ticketSpawnCmd.Flags().BoolVar(&ticketSpawnJSONFlag, "json", false, "Output as JSON")
	ticketSpawnCmd.Flags().BoolVar(&ticketSpawnResumeFlag, "resume", false, "Resume an orphaned session")
	ticketSpawnCmd.Flags().BoolVar(&ticketSpawnFreshFlag, "fresh", false, "Clear existing session and start fresh")
}
