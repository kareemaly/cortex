package api

import (
	"strings"

	"github.com/kareemaly/cortex/internal/ticket"
	"github.com/kareemaly/cortex/internal/types"
)

// Re-export shared types for API consumers
type (
	ErrorResponse            = types.ErrorResponse
	DatesResponse            = types.DatesResponse
	CommentResponse          = types.CommentResponse
	StatusEntryResponse      = types.StatusEntryResponse
	RequestedReviewResponse  = types.RequestedReviewResponse
	SessionResponse          = types.SessionResponse
	TicketResponse           = types.TicketResponse
	TicketSummary            = types.TicketSummary
	ListTicketsResponse      = types.ListTicketsResponse
	ListAllTicketsResponse   = types.ListAllTicketsResponse
	ArchitectSessionResponse = types.ArchitectSessionResponse
	ArchitectStateResponse   = types.ArchitectStateResponse
	ArchitectSpawnResponse   = types.ArchitectSpawnResponse
)

// CreateTicketRequest is the request body for creating a ticket.
type CreateTicketRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// UpdateTicketRequest is the request body for updating a ticket.
type UpdateTicketRequest struct {
	Title *string `json:"title,omitempty"`
	Body  *string `json:"body,omitempty"`
}

// MoveTicketRequest is the request body for moving a ticket.
type MoveTicketRequest struct {
	To string `json:"to"`
}

// SpawnResponse is the response for the spawn endpoint.
type SpawnResponse struct {
	Session SessionResponse `json:"session"`
	Ticket  TicketResponse  `json:"ticket"`
}

// AddCommentRequest is the request body for adding a comment to a ticket.
type AddCommentRequest struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// AddCommentResponse is the response for adding a comment.
type AddCommentResponse struct {
	Success bool            `json:"success"`
	Comment CommentResponse `json:"comment"`
}

// RequestReviewRequest is the request body for requesting a review.
type RequestReviewRequest struct {
	RepoPath string `json:"repo_path"`
	Summary  string `json:"summary"`
}

// RequestReviewResponse is the response for requesting a review.
type RequestReviewResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	ReviewCount int    `json:"review_count"`
}

// ConcludeSessionRequest is the request body for concluding a session.
type ConcludeSessionRequest struct {
	FullReport string `json:"full_report"`
}

// ConcludeSessionResponse is the response for concluding a session.
type ConcludeSessionResponse struct {
	Success  bool   `json:"success"`
	TicketID string `json:"ticket_id"`
	Message  string `json:"message"`
}

// filterSummaryList converts tickets to summaries with optional query filter.
// Query is matched case-insensitively against title or body.
func filterSummaryList(tickets []*ticket.Ticket, status ticket.Status, query string) []TicketSummary {
	var summaries []TicketSummary
	for _, t := range tickets {
		// Apply query filter if specified
		if query != "" &&
			!strings.Contains(strings.ToLower(t.Title), query) &&
			!strings.Contains(strings.ToLower(t.Body), query) {
			continue
		}
		summary := types.ToTicketSummary(t, status, true)
		summaries = append(summaries, summary)
	}
	if summaries == nil {
		summaries = []TicketSummary{}
	}
	return summaries
}
