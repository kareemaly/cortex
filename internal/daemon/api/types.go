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
	CommentResponse          = types.CommentResponse
	SessionResponse          = types.SessionResponse
	TicketResponse           = types.TicketResponse
	TicketSummary            = types.TicketSummary
	ListTicketsResponse      = types.ListTicketsResponse
	ListAllTicketsResponse   = types.ListAllTicketsResponse
	ArchitectSessionResponse = types.ArchitectSessionResponse
	ArchitectStateResponse   = types.ArchitectStateResponse
	ArchitectSpawnResponse   = types.ArchitectSpawnResponse
	DocResponse              = types.DocResponse
	DocSummary               = types.DocSummary
	ListDocsResponse         = types.ListDocsResponse
)

// CreateTicketRequest is the request body for creating a ticket.
type CreateTicketRequest struct {
	Title      string   `json:"title"`
	Body       string   `json:"body"`
	Type       string   `json:"type,omitempty"`
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

// AddCommentRequest is the request body for adding a comment to a ticket.
type AddCommentRequest struct {
	Type    string                       `json:"type"`
	Content string                       `json:"content"`
	Action  *types.CommentActionResponse `json:"action,omitempty"`
}

// AddCommentResponse is the response for adding a comment.
type AddCommentResponse struct {
	Success bool            `json:"success"`
	Comment CommentResponse `json:"comment"`
}

// RequestReviewRequest is the request body for requesting a review.
type RequestReviewRequest struct {
	RepoPath string `json:"repo_path"`
	Content  string `json:"content"`
	Commit   string `json:"commit,omitempty"`
}

// RequestReviewResponse is the response for requesting a review.
type RequestReviewResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Comment CommentResponse `json:"comment"`
}

// ConcludeSessionRequest is the request body for concluding a session.
type ConcludeSessionRequest struct {
	Content string `json:"content"`
}

// ConcludeSessionResponse is the response for concluding a session.
type ConcludeSessionResponse struct {
	Success  bool   `json:"success"`
	TicketID string `json:"ticket_id"`
	Message  string `json:"message"`
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

// CreateDocRequest is the request body for creating a doc.
type CreateDocRequest struct {
	Title      string   `json:"title"`
	Category   string   `json:"category"`
	Body       string   `json:"body,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	References []string `json:"references,omitempty"`
}

// UpdateDocRequest is the request body for updating a doc.
type UpdateDocRequest struct {
	Title      *string   `json:"title,omitempty"`
	Body       *string   `json:"body,omitempty"`
	Tags       *[]string `json:"tags,omitempty"`
	References *[]string `json:"references,omitempty"`
}

// MoveDocRequest is the request body for moving a doc.
type MoveDocRequest struct {
	Category string `json:"category"`
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
