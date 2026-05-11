package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/cli/tui/detail"
	"github.com/spf13/cobra"
)

var ticketShowCmd = &cobra.Command{
	Use:   "show <ticket-id>",
	Short: "Open a read-only ticket detail viewer",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ensureDaemon()

		architectPath, err := resolveArchitectPath("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client := sdk.DefaultClient(architectPath)
		ticketID := args[0]

		initial, err := loadTicketDetail(client, ticketID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		var program *tea.Program
		model := detail.New(
			initial.Title,
			initial.Subtitle,
			initial.Tabs,
		detail.WithEditableTicket(ticketID, initial.FilePath, func() tea.Msg {
			if err := program.ReleaseTerminal(); err != nil {
				return detail.EditFinished(detail.EditResult{}, err)
			}

			editErr := openEditor(initial.FilePath)

				restoreErr := program.RestoreTerminal()
				if editErr != nil {
					return detail.EditFinished(detail.EditResult{}, editErr)
				}
				if restoreErr != nil {
					return detail.EditFinished(detail.EditResult{}, restoreErr)
				}

				updated, err := loadTicketDetail(client, ticketID)
				if err != nil {
					return detail.EditFinished(detail.EditResult{}, err)
				}

				initial = updated
				return detail.EditFinished(detail.EditResult{
					Title:    updated.Title,
					Subtitle: updated.Subtitle,
					Tabs:     updated.Tabs,
				}, nil)
			}),
			detail.WithChangesLoader(func() tea.Msg {
				diffsResp, err := client.GetTicketDiffs(ticketID)
				if err != nil {
					return detail.ChangesLoaded(nil, err)
				}
				return detail.ChangesLoaded(buildChangesData(diffsResp), nil)
			}),
		)
		program = tea.NewProgram(model, tea.WithAltScreen())
		if _, err := program.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	ticketCmd.AddCommand(ticketShowCmd)
}

func fetchTicketSummary(client *sdk.Client, ticketResp *sdk.TicketResponse) (*sdk.TicketSummary, error) {
	listResp, err := client.ListTicketsByStatus(ticketResp.Status, "", nil)
	if err != nil {
		return nil, err
	}
	for i := range listResp.Tickets {
		if listResp.Tickets[i].ID == ticketResp.ID {
			return &listResp.Tickets[i], nil
		}
	}
	return nil, nil
}

type ticketDetailData struct {
	Title     string
	Subtitle  string
	Tabs      []detail.Tab
	FilePath  string
}

func loadTicketDetail(client *sdk.Client, ticketID string) (ticketDetailData, error) {
	ticketResp, err := client.GetTicketByID(ticketID)
	if err != nil {
		return ticketDetailData{}, err
	}

	ticketSummary, _ := fetchTicketSummary(client, ticketResp)

	var conclusionResp *sdk.ConclusionResponse
	var conclusionWarning string
	if ticketResp.HasConclusion {
		conclusionResp, err = client.GetConclusion(ticketResp.ID)
		if err != nil {
			conclusionWarning = err.Error()
		}
	}

	return ticketDetailData{
		Title:    ticketResp.Title,
		Tabs:     buildTicketTabs(ticketResp, ticketSummary, conclusionResp, conclusionWarning),
		FilePath: ticketResp.FilePath,
	}, nil
}

func openEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	cmd := exec.Command("sh", "-c", "exec "+editor+` "$1"`, "sh", path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func buildTicketTabs(ticketResp *sdk.TicketResponse, ticketSummary *sdk.TicketSummary, conclusionResp *sdk.ConclusionResponse, conclusionWarning string) []detail.Tab {
	tabs := []detail.Tab{
		{Label: "Overview", Content: buildTicketOverview(ticketResp, ticketSummary, conclusionResp, conclusionWarning), Kind: detail.TabKindMarkdown},
		{Label: "Ticket Body", Content: bodyContent(ticketResp.Body, "ticket"), Kind: detail.TabKindMarkdown},
	}
	if conclusionResp != nil || conclusionWarning != "" {
		tabs = append(tabs, detail.Tab{Label: "Conclusion", Content: buildLinkedConclusionTab(ticketResp, conclusionResp, conclusionWarning), Kind: detail.TabKindMarkdown})
	}
	if conclusionResp != nil && len(conclusionResp.Commits) > 0 {
		tabs = append(tabs, detail.Tab{Label: "Changes", Kind: detail.TabKindChanges})
	}
	return tabs
}

func buildTicketOverview(ticketResp *sdk.TicketResponse, ticketSummary *sdk.TicketSummary, conclusionResp *sdk.ConclusionResponse, conclusionWarning string) string {
	var b strings.Builder
	b.WriteString("## Ticket\n")
	b.WriteString(fmt.Sprintf("- ID: `%s`\n", ticketResp.ID))
	b.WriteString(fmt.Sprintf("- Status: `%s`\n", ticketResp.Status))
	b.WriteString(fmt.Sprintf("- Repo: `%s`\n", emptyDash(ticketResp.Repo)))
	b.WriteString(fmt.Sprintf("- Path: `%s`\n", emptyDash(ticketResp.Path)))
	b.WriteString(fmt.Sprintf("- Created: %s\n", formatDetailTime(ticketResp.Created)))
	b.WriteString(fmt.Sprintf("- Updated: %s\n", formatDetailTime(ticketResp.Updated)))
	b.WriteString(fmt.Sprintf("- Due: %s\n", formatDetailOptionalTime(ticketResp.Due)))

	b.WriteString("\n## References\n")
	b.WriteString(markdownList(ticketResp.References, "- none"))
	b.WriteString("\n")

	b.WriteString("\n## Active Session\n")
	if ticketSummary == nil {
		b.WriteString("- State: unavailable\n")
	} else if !ticketSummary.HasActiveSession {
		b.WriteString("- Active: no\n")
	} else {
		b.WriteString("- Active: yes\n")
		b.WriteString(fmt.Sprintf("- Agent: `%s`\n", emptyDash(ticketSummary.Agent)))
		if ticketSummary.AgentStatus != nil {
			b.WriteString(fmt.Sprintf("- Status: `%s`\n", *ticketSummary.AgentStatus))
		}
		if ticketSummary.AgentTool != nil {
			b.WriteString(fmt.Sprintf("- Tool: `%s`\n", *ticketSummary.AgentTool))
		}
		b.WriteString(fmt.Sprintf("- Orphaned: %s\n", yesNo(ticketSummary.IsOrphaned)))
		b.WriteString(fmt.Sprintf("- Started: %s\n", formatDetailOptionalTime(ticketSummary.SessionStartedAt)))
	}

	b.WriteString("\n## Linked Conclusion\n")
	if !ticketResp.HasConclusion {
		b.WriteString("- Linked conclusion: none\n")
	} else {
		b.WriteString(fmt.Sprintf("- ID: `%s`\n", ticketResp.ID))
		if conclusionResp != nil {
			b.WriteString(fmt.Sprintf("- Agent: `%s`\n", emptyDash(conclusionResp.Agent)))
			b.WriteString(fmt.Sprintf("- Started: %s\n", formatDetailTime(conclusionResp.StartedAt)))
			b.WriteString(fmt.Sprintf("- Concluded: %s\n", formatDetailTime(conclusionResp.ConcludedAt)))
			b.WriteString(fmt.Sprintf("- Rejected: %s\n", yesNo(conclusionResp.Rejected)))
			if conclusionResp.RejectionReason != "" {
				b.WriteString(fmt.Sprintf("- Rejection reason: %s\n", conclusionResp.RejectionReason))
			}
			if len(conclusionResp.Commits) > 0 {
				b.WriteString(fmt.Sprintf("- Commits: %d\n", len(conclusionResp.Commits)))
			}
		} else if conclusionWarning != "" {
			b.WriteString(fmt.Sprintf("- Load error: %s\n", conclusionWarning))
		}
	}

	return b.String()
}

func buildLinkedConclusionTab(ticketResp *sdk.TicketResponse, conclusionResp *sdk.ConclusionResponse, conclusionWarning string) string {
	if conclusionResp == nil {
		return fmt.Sprintf("Unable to load linked conclusion `%s`.\n\n%s", ticketResp.ID, conclusionWarning)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("- ID: `%s`\n", conclusionResp.ID))
	b.WriteString(fmt.Sprintf("- Agent: `%s`\n", emptyDash(conclusionResp.Agent)))
	b.WriteString(fmt.Sprintf("- Profile: `%s`\n", emptyDash(conclusionResp.Profile)))
	b.WriteString(fmt.Sprintf("- Started: %s\n", formatDetailTime(conclusionResp.StartedAt)))
	b.WriteString(fmt.Sprintf("- Concluded: %s\n", formatDetailTime(conclusionResp.ConcludedAt)))
	b.WriteString(fmt.Sprintf("- Duration: %s\n", formatDetailDuration(conclusionResp.StartedAt, conclusionResp.ConcludedAt)))
	b.WriteString(fmt.Sprintf("- Rejected: %s\n", yesNo(conclusionResp.Rejected)))
	if conclusionResp.RejectionReason != "" {
		b.WriteString(fmt.Sprintf("- Rejection reason: %s\n", conclusionResp.RejectionReason))
	}

	b.WriteString("\n## Commits\n")
	b.WriteString(markdownList(conclusionResp.Commits, "- none"))
	b.WriteString("\n\n## Body\n\n")
	b.WriteString(bodyContent(conclusionResp.Body, "conclusion"))

	return b.String()
}

func bodyContent(content, label string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return fmt.Sprintf("_Empty %s body._", label)
	}
	return trimmed
}

func emptyDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func buildChangesData(resp *sdk.DiffsResponse) *detail.ChangesData {
	if resp == nil {
		return nil
	}

	changes := &detail.ChangesData{
		Repo:    resp.Repo,
		Commits: make([]detail.ChangeCommit, 0, len(resp.Commits)),
	}

	for _, commit := range resp.Commits {
		files := make([]detail.ChangeFile, 0, len(commit.Files))
		for _, file := range commit.Files {
			files = append(files, detail.ChangeFile{
				Path:      file.Path,
				OldPath:   stringOrEmpty(file.OldPath),
				Status:    file.Status,
				IsBinary:  file.IsBinary,
				Additions: file.Additions,
				Deletions: file.Deletions,
				Patch:     file.Patch,
			})
		}

		changes.Commits = append(changes.Commits, detail.ChangeCommit{
			SHA:        commit.SHA,
			Subject:    commit.Subject,
			AuthorName: commit.AuthorName,
			AuthoredAt: commit.AuthoredAt,
			Files:      files,
		})
	}

	return changes
}

func stringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
