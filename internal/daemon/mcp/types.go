package mcp

import (
	"time"

	"github.com/kareemaly/cortex/internal/ticket"
	"github.com/kareemaly/cortex/internal/types"
)

// SessionType indicates the type of MCP session.
type SessionType string

const (
	SessionTypeArchitect SessionType = "architect"
	SessionTypeTicket    SessionType = "ticket"
)

// Session holds the current session context.
type Session struct {
	Type     SessionType
	TicketID string // Only set for ticket sessions
}

// Input types for architect tools

// ListTicketsInput is the input for the listTickets tool.
type ListTicketsInput struct {
	Status string `json:"status" jsonschema:"Ticket status to filter by (required). Must be one of: backlog, progress, review, done"`
	Query  string `json:"query,omitempty" jsonschema:"Optional search term to filter tickets by title/body (case-insensitive substring match)."`
}

// ReadTicketInput is the input for the readTicket tool.
type ReadTicketInput struct {
	ID string `json:"id" jsonschema:"The ticket ID to read"`
}

// CreateTicketInput is the input for the createTicket tool.
type CreateTicketInput struct {
	Title string `json:"title" jsonschema:"The ticket title (required)"`
	Body  string `json:"body,omitempty" jsonschema:"The ticket body/description"`
}

// UpdateTicketInput is the input for the updateTicket tool.
type UpdateTicketInput struct {
	ID    string  `json:"id" jsonschema:"The ticket ID to update"`
	Title *string `json:"title,omitempty" jsonschema:"New title (optional)"`
	Body  *string `json:"body,omitempty" jsonschema:"New body (optional)"`
}

// DeleteTicketInput is the input for the deleteTicket tool.
type DeleteTicketInput struct {
	ID string `json:"id" jsonschema:"The ticket ID to delete"`
}

// MoveTicketInput is the input for the moveTicket tool.
type MoveTicketInput struct {
	ID     string `json:"id" jsonschema:"The ticket ID to move"`
	Status string `json:"status" jsonschema:"Target status (backlog/progress/review/done)"`
}

// SpawnSessionInput is the input for the spawnSession tool.
type SpawnSessionInput struct {
	TicketID string `json:"ticket_id" jsonschema:"The ticket ID to spawn a session for"`
	Agent    string `json:"agent,omitempty" jsonschema:"Agent name (default: claude)"`
	Mode     string `json:"mode,omitempty" jsonschema:"Spawn mode: 'normal' (default), 'resume', or 'fresh'"`
}

// Input types for ticket tools

// AddCommentInput is the input for the addTicketComment tool.
type AddCommentInput struct {
	Type    string `json:"type" jsonschema:"Comment type (scope_change/decision/blocker/progress/question/rejection/general/ticket_done)"`
	Content string `json:"content" jsonschema:"The comment content"`
}

// RequestReviewInput is the input for the requestReview tool.
type RequestReviewInput struct {
	RepoPath string `json:"repo_path" jsonschema:"Path to the repository being reviewed"`
	Summary  string `json:"summary" jsonschema:"Summary of changes for the reviewer"`
}

// ConcludeSessionInput is the input for the concludeSession tool.
type ConcludeSessionInput struct {
	FullReport string `json:"full_report" jsonschema:"Complete summary of work done, decisions made, and files changed"`
}

// Type aliases for identical types (map to shared types)
type (
	DatesOutput  = types.DatesResponse
	StatusOutput = types.StatusEntryResponse
)

// CommentOutput represents a comment on a ticket.
// Alias to shared type for JSON compatibility.
type CommentOutput = types.CommentResponse

// MCP-specific output types (structurally different from shared types)

// TicketSummary is a brief ticket representation for list views.
// MCP version is simpler: no Updated, AgentStatus, AgentTool fields.
type TicketSummary struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Status           string    `json:"status"`
	Created          time.Time `json:"created"`
	HasActiveSession bool      `json:"has_active_session"`
}

// SessionOutput represents a work session.
// MCP version has IsActive but no StatusHistory/RequestedReviews.
type SessionOutput struct {
	ID            string        `json:"id"`
	StartedAt     time.Time     `json:"started_at"`
	EndedAt       *time.Time    `json:"ended_at,omitempty"`
	Agent         string        `json:"agent"`
	TmuxWindow    string        `json:"tmux_window"`
	CurrentStatus *StatusOutput `json:"current_status,omitempty"`
	IsActive      bool          `json:"is_active"`
}

// TicketOutput is the full ticket representation.
// Uses MCP-specific SessionOutput.
type TicketOutput struct {
	ID       string          `json:"id"`
	Title    string          `json:"title"`
	Body     string          `json:"body"`
	Status   string          `json:"status"`
	Dates    DatesOutput     `json:"dates"`
	Comments []CommentOutput `json:"comments"`
	Session  *SessionOutput  `json:"session,omitempty"`
}

// Tool output wrappers

// ListTicketsOutput is the output for the listTickets tool.
type ListTicketsOutput struct {
	Tickets []TicketSummary `json:"tickets"`
	Total   int             `json:"total"`
}

// ReadTicketOutput is the output for the readTicket tool.
type ReadTicketOutput struct {
	Ticket TicketOutput `json:"ticket"`
}

// CreateTicketOutput is the output for the createTicket tool.
type CreateTicketOutput struct {
	Ticket TicketOutput `json:"ticket"`
}

// UpdateTicketOutput is the output for the updateTicket tool.
type UpdateTicketOutput struct {
	Ticket TicketOutput `json:"ticket"`
}

// DeleteTicketOutput is the output for the deleteTicket tool.
type DeleteTicketOutput struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
}

// MoveTicketOutput is the output for the moveTicket tool.
type MoveTicketOutput struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Status  string `json:"status"`
}

// SpawnSessionOutput is the output for the spawnSession tool.
type SpawnSessionOutput struct {
	Success    bool   `json:"success"`
	TicketID   string `json:"ticket_id,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
	TmuxWindow string `json:"tmux_window,omitempty"`
	State      string `json:"state,omitempty"`
	Message    string `json:"message,omitempty"`
}

// AddCommentOutput is the output for the addTicketComment tool.
type AddCommentOutput struct {
	Success bool          `json:"success"`
	Comment CommentOutput `json:"comment,omitempty"`
}

// RequestReviewOutput is the output for the requestReview tool.
type RequestReviewOutput struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	ReviewCount int    `json:"review_count"`
}

// ConcludeSessionOutput is the output for the concludeSession tool.
type ConcludeSessionOutput struct {
	Success  bool   `json:"success"`
	TicketID string `json:"ticket_id"`
	Message  string `json:"message,omitempty"`
}

// Conversion functions

// ToTicketOutput converts a ticket and status to output format.
func ToTicketOutput(t *ticket.Ticket, status ticket.Status) TicketOutput {
	var session *SessionOutput
	if t.Session != nil {
		s := ToSessionOutput(t.Session)
		session = &s
	}

	comments := make([]CommentOutput, len(t.Comments))
	for i, c := range t.Comments {
		comments[i] = types.ToCommentResponse(&c)
	}

	return TicketOutput{
		ID:       t.ID,
		Title:    t.Title,
		Body:     t.Body,
		Status:   string(status),
		Dates:    types.ToDatesResponse(t.Dates),
		Comments: comments,
		Session:  session,
	}
}

// ToTicketSummary converts a ticket and status to summary format.
func ToTicketSummary(t *ticket.Ticket, status ticket.Status) TicketSummary {
	return TicketSummary{
		ID:               t.ID,
		Title:            t.Title,
		Status:           string(status),
		Created:          t.Dates.Created,
		HasActiveSession: t.HasActiveSession(),
	}
}

// ToSessionOutput converts a session to output format.
func ToSessionOutput(s *ticket.Session) SessionOutput {
	var currentStatus *StatusOutput
	if s.CurrentStatus != nil {
		cs := types.ToStatusEntryResponse(*s.CurrentStatus)
		currentStatus = &cs
	}

	return SessionOutput{
		ID:            s.ID,
		StartedAt:     s.StartedAt,
		EndedAt:       s.EndedAt,
		Agent:         s.Agent,
		TmuxWindow:    s.TmuxWindow,
		CurrentStatus: currentStatus,
		IsActive:      s.IsActive(),
	}
}
