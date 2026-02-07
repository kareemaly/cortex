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

var ticketShowJSONFlag bool

var ticketShowCmd = &cobra.Command{
	Use:   "<id>",
	Short: "Show ticket details",
	Long:  `Show detailed information about a ticket, including its session and comments.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ensureDaemon()

		projectPath, err := resolveProjectPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		ticketID := args[0]
		client := sdk.DefaultClient(projectPath)

		// JSON output: fetch and print.
		if ticketShowJSONFlag {
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
	ticketShowCmd.Flags().BoolVar(&ticketShowJSONFlag, "json", false, "Output as JSON")
}

// printTicketPlainText fetches and prints ticket details in plain text format.
func printTicketPlainText(client *sdk.Client, ticketID string) {
	t, err := client.FindTicketByID(ticketID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print ticket info.
	fmt.Printf("Ticket: %s\n", t.ID)
	fmt.Printf("  Title:  %s\n", t.Title)
	fmt.Printf("  Status: %s\n", t.Status)
	fmt.Printf("  Type:   %s\n", t.Type)
	fmt.Printf("  Created: %s\n", t.Created.Format("Jan 02 15:04:05"))
	fmt.Printf("  Updated: %s\n", t.Updated.Format("Jan 02 15:04:05"))
	fmt.Println()

	// Print body.
	if t.Body != "" {
		fmt.Println("Body:")
		fmt.Println(t.Body)
		fmt.Println()
	}

	// Print comments.
	if len(t.Comments) > 0 {
		fmt.Println("Comments:")
		for _, c := range t.Comments {
			fmt.Printf("  [%s] %s\n", c.Type, c.Content)
		}
	}
}
