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
	ticketListStatus   string
	ticketListQuery    string
	ticketListJSONFlag bool
)

var ticketListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tickets",
	Long:  `List tickets, optionally filtered by status or search query.`,
	Run: func(cmd *cobra.Command, args []string) {
		ensureDaemon()

		projectPath, err := resolveProjectPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client := sdk.DefaultClient(projectPath)

		if ticketListStatus != "" {
			ticketListByStatus(client, ticketListStatus, ticketListQuery)
		} else {
			ticketListAll(client, ticketListQuery)
		}
	},
}

func init() {
	ticketListCmd.Flags().StringVar(&ticketListStatus, "status", "", "Filter tickets by status (backlog, progress, done)")
	ticketListCmd.Flags().StringVar(&ticketListQuery, "query", "", "Filter tickets by title or body (case-insensitive)")
	ticketListCmd.Flags().BoolVar(&ticketListJSONFlag, "json", false, "Output as JSON")
}

func ticketListAll(client *sdk.Client, query string) {
	resp, err := client.ListAllTickets(query, nil, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if ticketListJSONFlag {
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
		ticketPrintTable(w, resp.Backlog)
		_, _ = fmt.Fprintln(w)
	}

	// Print progress
	if len(resp.Progress) > 0 {
		_, _ = fmt.Fprintln(w, "PROGRESS")
		ticketPrintTable(w, resp.Progress)
		_, _ = fmt.Fprintln(w)
	}

	// Print done
	if len(resp.Done) > 0 {
		_, _ = fmt.Fprintln(w, "DONE")
		ticketPrintTable(w, resp.Done)
		_, _ = fmt.Fprintln(w)
	}

	_ = w.Flush()

	total := len(resp.Backlog) + len(resp.Progress) + len(resp.Done)
	if total == 0 {
		fmt.Println("No tickets found.")
	}
}

func ticketListByStatus(client *sdk.Client, status, query string) {
	resp, err := client.ListTicketsByStatus(status, query, nil, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if ticketListJSONFlag {
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
	ticketPrintTable(w, resp.Tickets)
	_ = w.Flush()
}

func ticketPrintTable(w *tabwriter.Writer, tickets []sdk.TicketSummary) {
	_, _ = fmt.Fprintln(w, "ID\tTYPE\tTITLE\tCREATED\tACTIVE")
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
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", shortID, t.Type, title, ticketFormatTime(t.Created), active)
	}
}

func ticketFormatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("Jan 02 15:04")
}
