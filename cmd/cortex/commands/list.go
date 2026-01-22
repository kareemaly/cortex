package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/spf13/cobra"
)

var (
	listStatus   string
	listJSONFlag bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tickets",
	Run: func(cmd *cobra.Command, args []string) {
		projectPath, err := resolveProjectPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client := sdk.DefaultClient(projectPath)

		if listStatus != "" {
			listByStatus(client, listStatus)
		} else {
			listAll(client)
		}
	},
}

func init() {
	listCmd.Flags().StringVar(&listStatus, "status", "", "Filter tickets by status (backlog, progress, done)")
	listCmd.Flags().BoolVar(&listJSONFlag, "json", false, "Output as JSON")
	rootCmd.AddCommand(listCmd)
}

func listAll(client *sdk.Client) {
	resp, err := client.ListAllTickets()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if listJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(resp); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print backlog
	if len(resp.Backlog) > 0 {
		_, _ = fmt.Fprintln(w, "BACKLOG")
		printTicketTable(w, resp.Backlog)
		_, _ = fmt.Fprintln(w)
	}

	// Print progress
	if len(resp.Progress) > 0 {
		_, _ = fmt.Fprintln(w, "PROGRESS")
		printTicketTable(w, resp.Progress)
		_, _ = fmt.Fprintln(w)
	}

	// Print done
	if len(resp.Done) > 0 {
		_, _ = fmt.Fprintln(w, "DONE")
		printTicketTable(w, resp.Done)
		_, _ = fmt.Fprintln(w)
	}

	_ = w.Flush()

	total := len(resp.Backlog) + len(resp.Progress) + len(resp.Done)
	if total == 0 {
		fmt.Println("No tickets found.")
	}
}

func listByStatus(client *sdk.Client, status string) {
	resp, err := client.ListTicketsByStatus(status)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if listJSONFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(resp); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if len(resp.Tickets) == 0 {
		fmt.Printf("No tickets in %s.\n", status)
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	printTicketTable(w, resp.Tickets)
	_ = w.Flush()
}

func printTicketTable(w *tabwriter.Writer, tickets []sdk.TicketSummary) {
	_, _ = fmt.Fprintln(w, "ID\tTITLE\tCREATED\tACTIVE")
	for _, t := range tickets {
		shortID := t.ID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		title := t.Title
		if len(title) > 40 {
			title = title[:37] + "..."
		}
		active := ""
		if t.HasActiveSession {
			active = "yes"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", shortID, title, formatTime(t.Created), active)
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("Jan 02 15:04")
}
