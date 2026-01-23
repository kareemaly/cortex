package commands

import (
	"encoding/json"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/cli/tui/ticket"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var showJSONFlag bool

var showCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Show ticket details",
	Long: `Show detailed information about a ticket, including its session and comments.

If no ticket ID is provided, the command will look for the CORTEX_TICKET_ID
environment variable (set automatically in spawned ticket sessions).`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectPath, err := resolveProjectPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Get ticket ID from args or environment variable
		var ticketID string
		if len(args) > 0 {
			ticketID = args[0]
		} else {
			ticketID = os.Getenv("CORTEX_TICKET_ID")
		}

		if ticketID == "" {
			fmt.Fprintf(os.Stderr, "Error: ticket ID required (provide as argument or set CORTEX_TICKET_ID)\n")
			os.Exit(1)
		}

		client := sdk.DefaultClient(projectPath)

		// JSON output: fetch and print.
		if showJSONFlag {
			t, err := client.FindTicketByID(ticketID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(t); err != nil {
				fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// Non-TTY: print plain text output.
		if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
			printTicketPlainText(client, ticketID)
			return
		}

		// TTY: launch interactive TUI.
		model := ticket.New(client, ticketID)
		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	showCmd.Flags().BoolVar(&showJSONFlag, "json", false, "Output as JSON")
	rootCmd.AddCommand(showCmd)
}
