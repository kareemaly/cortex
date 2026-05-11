package commands

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/cli/tui/detail"
	"github.com/spf13/cobra"
)

var conclusionShowCmd = &cobra.Command{
	Use:   "show <conclusion-id>",
	Short: "Open a read-only conclusion detail viewer",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ensureDaemon()

		architectPath, err := resolveArchitectPath("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client := sdk.DefaultClient(architectPath)
		conclusionResp, err := client.GetConclusion(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		tabs := []detail.Tab{
			{Label: "Overview", Content: buildConclusionOverview(conclusionResp)},
			{Label: "Body", Content: bodyContent(conclusionResp.Body, "conclusion")},
		}

		program := tea.NewProgram(detail.New(conclusionTitle(conclusionResp), "", tabs), tea.WithAltScreen())
		if _, err := program.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	conclusionCmd.AddCommand(conclusionShowCmd)
}

func buildConclusionOverview(conclusionResp *sdk.ConclusionResponse) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("- ID: `%s`\n", conclusionResp.ID))
	b.WriteString(fmt.Sprintf("- Agent: `%s`\n", emptyDash(conclusionResp.Agent)))
	b.WriteString(fmt.Sprintf("- Profile: `%s`\n", emptyDash(conclusionResp.Profile)))
	b.WriteString(fmt.Sprintf("- Ticket: `%s`\n", emptyDash(conclusionResp.TicketID)))
	b.WriteString(fmt.Sprintf("- Started: %s\n", formatDetailTime(conclusionResp.StartedAt)))
	b.WriteString(fmt.Sprintf("- Concluded: %s\n", formatDetailTime(conclusionResp.ConcludedAt)))
	b.WriteString(fmt.Sprintf("- Duration: %s\n", formatDetailDuration(conclusionResp.StartedAt, conclusionResp.ConcludedAt)))
	b.WriteString(fmt.Sprintf("- Rejected: %s\n", yesNo(conclusionResp.Rejected)))
	if conclusionResp.RejectionReason != "" {
		b.WriteString(fmt.Sprintf("- Rejection reason: %s\n", conclusionResp.RejectionReason))
	}

	if conclusionResp.CollabID != "" {
		b.WriteString(fmt.Sprintf("- Collab ID: `%s`\n", conclusionResp.CollabID))
	}

	b.WriteString("\n## Commits\n")
	b.WriteString(markdownList(conclusionResp.Commits, "- none"))

	return b.String()
}

func conclusionTitle(conclusionResp *sdk.ConclusionResponse) string {
	if conclusionResp.TicketID != "" {
		return fmt.Sprintf("Ticket Conclusion %s", shortID(conclusionResp.TicketID))
	}
	if conclusionResp.CollabID != "" {
		return fmt.Sprintf("Collab %s", shortID(conclusionResp.CollabID))
	}
	return "Conclusion"
}
