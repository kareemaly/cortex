package api

import (
	"strings"
	"time"

	"github.com/kareemaly/cortex/internal/session"
	"github.com/kareemaly/cortex/internal/ticket"
	"github.com/kareemaly/cortex/internal/types"
)

type (
	ErrorResponse            = types.ErrorResponse
	SessionResponse          = types.SessionResponse
	TicketResponse           = types.TicketResponse
	TicketSummary            = types.TicketSummary
	ListTicketsResponse      = types.ListTicketsResponse
	ListAllTicketsResponse   = types.ListAllTicketsResponse
	DiffFileResponse         = types.DiffFileResponse
	CommitDiffResponse       = types.CommitDiffResponse
	DiffsResponse            = types.DiffsResponse
	ArchitectSessionResponse = types.ArchitectSessionResponse
	ArchitectStateResponse   = types.ArchitectStateResponse
	ArchitectSpawnResponse   = types.ArchitectSpawnResponse
	ConclusionResponse       = types.ConclusionResponse
	ListConclusionsResponse  = types.ListConclusionsResponse
	HealthResponse           = types.HealthResponse
	ArchitectTicketCounts    = types.ArchitectTicketCounts
	ArchitectResponse        = types.ArchitectResponse
	ConcludeSessionResponse  = types.ConcludeSessionResponse
	ResolvePromptResponse    = types.ResolvePromptResponse
	PromptFileInfo           = types.PromptFileInfo
	PromptGroupInfo          = types.PromptGroupInfo
	ListPromptsResponse      = types.ListPromptsResponse
	SpawnCollabResponse      = types.SpawnCollabResponse
)

type CreateTicketRequest struct {
	Title      string   `json:"title"`
	Body       string   `json:"body"`
	Repo       string   `json:"repo,omitempty"`
	Path       string   `json:"path,omitempty"`
	DueDate    *string  `json:"due_date,omitempty"`
	References []string `json:"references,omitempty"`
}

type UpdateTicketRequest struct {
	Title      *string   `json:"title,omitempty"`
	Body       *string   `json:"body,omitempty"`
	References *[]string `json:"references,omitempty"`
}

type EditTicketBodyRequest struct {
	OldString  string `json:"oldString"`
	NewString  string `json:"newString"`
	ReplaceAll bool   `json:"replaceAll,omitempty"`
}

type MoveTicketRequest struct {
	To string `json:"to"`
}

type SetDueDateRequest struct {
	DueDate string `json:"due_date"`
}

type SpawnResponse struct {
	Session SessionResponse `json:"session,omitempty"`
	Ticket  TicketResponse  `json:"ticket"`
}

type ConcludeSessionRequest struct {
	Content         string   `json:"content"`
	StartedAt       string   `json:"started_at,omitempty"`
	Commits         []string `json:"commits,omitempty"`
	Rejected        bool     `json:"rejected,omitempty"`
	RejectionReason string   `json:"rejection_reason,omitempty"`
}

type FocusResponse struct {
	Success bool   `json:"success"`
	Window  string `json:"window"`
}

type ExecuteActionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type EjectPromptRequest struct {
	Path string `json:"path"`
}

type EditPromptRequest struct {
	Path string `json:"path"`
}

type ResetPromptRequest struct {
	Path string `json:"path"`
}

type SpawnCollabRequest struct {
	Path    string `json:"path"`
	Prompt  string `json:"prompt"`
	Slug    string `json:"slug"`
	Mode    string `json:"mode,omitempty"`
	Variant string `json:"variant,omitempty"`
}

func filterSummaryList(tickets []*ticket.Ticket, status ticket.Status, query string, dueBefore *time.Time, tmuxSession string, checker types.TmuxChecker, sessionMgr *SessionManager, projectPath string, receiverMgr *ReceiverManager, ticketStore *ticket.Store) []TicketSummary {
	var summaries []TicketSummary

	var sessStore *session.Store
	if sessionMgr != nil {
		sessStore = sessionMgr.GetStore(projectPath)
	}

	for _, t := range tickets {
		if query != "" &&
			!strings.Contains(strings.ToLower(t.Title), query) &&
			!strings.Contains(strings.ToLower(t.Body), query) {
			continue
		}
		if dueBefore != nil {
			if t.Due == nil || !t.Due.Before(*dueBefore) {
				continue
			}
		}

		var sess *session.Session
		if sessStore != nil {
			sess, _ = sessStore.GetByTicketID(t.ID)
		}

		summary := types.ToTicketSummary(t, status, sess, tmuxSession, checker)

		hasConclusion := false
		if ticketStore != nil && status == ticket.StatusDone {
			if ok, err := ticketStore.HasConclusion(t.ID); err == nil && ok {
				hasConclusion = true
			}
		}
		summary.HasConclusion = hasConclusion

		if receiverMgr != nil && sess != nil && sess.SessionID != "" {
			if ev, ok := receiverMgr.GetEvent(sess.SessionID); ok {
				s := string(ev.Status)
				summary.AgentStatus = &s
				if ev.Tool != "" {
					summary.AgentTool = &ev.Tool
				} else {
					summary.AgentTool = nil
				}
			}
		}

		summaries = append(summaries, summary)
	}
	if summaries == nil {
		summaries = []TicketSummary{}
	}
	return summaries
}
