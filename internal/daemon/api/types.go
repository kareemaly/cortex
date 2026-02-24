package api

import (
	"strings"
	"time"

	"github.com/kareemaly/cortex/internal/session"
	"github.com/kareemaly/cortex/internal/ticket"
	"github.com/kareemaly/cortex/internal/types"
)

// Re-export shared types for API consumers
type (
	ErrorResponse            = types.ErrorResponse
	SessionResponse          = types.SessionResponse
	TicketResponse           = types.TicketResponse
	TicketSummary            = types.TicketSummary
	ListTicketsResponse      = types.ListTicketsResponse
	ListAllTicketsResponse   = types.ListAllTicketsResponse
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

// CreateTicketRequest is the request body for creating a ticket.
type CreateTicketRequest struct {
	Title      string   `json:"title"`
	Body       string   `json:"body"`
	Type       string   `json:"type,omitempty"`
	Repo       string   `json:"repo,omitempty"`
	Path       string   `json:"path,omitempty"`
	DueDate    *string  `json:"due_date,omitempty"`
	References []string `json:"references,omitempty"`
}

// UpdateTicketRequest is the request body for updating a ticket.
type UpdateTicketRequest struct {
	Title      *string   `json:"title,omitempty"`
	Body       *string   `json:"body,omitempty"`
	References *[]string `json:"references,omitempty"`
}

// MoveTicketRequest is the request body for moving a ticket.
type MoveTicketRequest struct {
	To string `json:"to"`
}

// SetDueDateRequest is the request body for setting a ticket's due date.
type SetDueDateRequest struct {
	DueDate string `json:"due_date"`
}

// SpawnResponse is the response for the spawn endpoint.
type SpawnResponse struct {
	Session SessionResponse `json:"session"`
	Ticket  TicketResponse  `json:"ticket"`
}

// ConcludeSessionRequest is the request body for concluding a session.
type ConcludeSessionRequest struct {
	Content   string `json:"content"`
	Type      string `json:"type,omitempty"`
	Repo      string `json:"repo,omitempty"`
	StartedAt string `json:"started_at,omitempty"`
}

// FocusResponse is the response for the focus endpoint.
type FocusResponse struct {
	Success bool   `json:"success"`
	Window  string `json:"window"`
}

// ExecuteActionResponse is the response for executing a comment action.
type ExecuteActionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// EjectPromptRequest is the request body for ejecting a prompt.
type EjectPromptRequest struct {
	Path string `json:"path"`
}

// EditPromptRequest is the request body for editing a prompt in $EDITOR.
type EditPromptRequest struct {
	Path string `json:"path"`
}

// ResetPromptRequest is the request body for resetting an ejected prompt to default.
type ResetPromptRequest struct {
	Path string `json:"path"`
}

// CreateConclusionRequest is the request body for creating a conclusion.
type CreateConclusionRequest struct {
	Type      string `json:"type"`
	Ticket    string `json:"ticket,omitempty"`
	Repo      string `json:"repo,omitempty"`
	Body      string `json:"body"`
	StartedAt string `json:"started_at,omitempty"`
}

// filterSummaryList converts tickets to summaries with optional query and dueBefore filters.
// Looks up session from session manager for each ticket.
func filterSummaryList(tickets []*ticket.Ticket, status ticket.Status, query string, dueBefore *time.Time, tmuxSession string, checker types.TmuxChecker, sessionMgr *SessionManager, projectPath string) []TicketSummary {
	var summaries []TicketSummary

	// Get the session store for this project
	var sessStore *session.Store
	if sessionMgr != nil {
		sessStore = sessionMgr.GetStore(projectPath)
	}

	for _, t := range tickets {
		// Apply query filter if specified
		if query != "" &&
			!strings.Contains(strings.ToLower(t.Title), query) &&
			!strings.Contains(strings.ToLower(t.Body), query) {
			continue
		}
		// Apply dueBefore filter if specified
		if dueBefore != nil {
			if t.Due == nil || !t.Due.Before(*dueBefore) {
				continue
			}
		}

		// Look up session for this ticket
		var sess *session.Session
		if sessStore != nil {
			sess, _ = sessStore.GetByTicketID(t.ID)
		}

		summary := types.ToTicketSummary(t, status, sess, tmuxSession, checker)
		summaries = append(summaries, summary)
	}
	if summaries == nil {
		summaries = []TicketSummary{}
	}
	return summaries
}
