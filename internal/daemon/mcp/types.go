package mcp

import (
	"time"

	"github.com/kareemaly/cortex/internal/ticket"
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
	Status string `json:"status,omitempty" jsonschema:"Filter by status (backlog/progress/review/done). Leave empty for all tickets."`
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
	Type    string `json:"type" jsonschema:"Comment type (scope_change/decision/blocker/progress/question/rejection/general)"`
	Content string `json:"content" jsonschema:"The comment content"`
}

// ConcludeSessionInput is the input for the concludeSession tool.
type ConcludeSessionInput struct {
	Summary string `json:"summary,omitempty" jsonschema:"Final summary of work done in this session"`
}

// SubmitReportInput is the input for the submitReport tool (deprecated).
type SubmitReportInput struct {
	Files        []string `json:"files,omitempty" jsonschema:"List of modified files"`
	ScopeChanges *string  `json:"scope_changes,omitempty" jsonschema:"Description of any scope changes"`
	Decisions    []string `json:"decisions,omitempty" jsonschema:"List of decisions made"`
	Summary      string   `json:"summary,omitempty" jsonschema:"Summary of work done"`
}

// ApproveInput is the input for the approve tool (deprecated).
type ApproveInput struct {
	Summary       string `json:"summary,omitempty" jsonschema:"Final summary before approval"`
	CommitMessage string `json:"commit_message,omitempty" jsonschema:"Commit message for on_approve hooks"`
}

// Output types

// HookResultOutput represents the result of executing a single hook.
type HookResultOutput struct {
	Command  string `json:"command"`
	Stdout   string `json:"stdout"`
	ExitCode int    `json:"exit_code"`
}

// HooksExecutionOutput represents the overall result of executing hooks.
type HooksExecutionOutput struct {
	Executed bool               `json:"executed"`
	Success  bool               `json:"success"`
	Hooks    []HookResultOutput `json:"hooks,omitempty"`
}

// TicketSummary is a brief ticket representation for list views.
type TicketSummary struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Status           string    `json:"status"`
	Created          time.Time `json:"created"`
	HasActiveSession bool      `json:"has_active_session"`
}

// TicketOutput is the full ticket representation.
type TicketOutput struct {
	ID       string          `json:"id"`
	Title    string          `json:"title"`
	Body     string          `json:"body"`
	Status   string          `json:"status"`
	Dates    DatesOutput     `json:"dates"`
	Comments []CommentOutput `json:"comments"`
	Session  *SessionOutput  `json:"session,omitempty"`
}

// DatesOutput represents ticket date information.
type DatesOutput struct {
	Created  time.Time  `json:"created"`
	Updated  time.Time  `json:"updated"`
	Progress *time.Time `json:"progress,omitempty"`
	Reviewed *time.Time `json:"reviewed,omitempty"`
	Done     *time.Time `json:"done,omitempty"`
}

// CommentOutput represents a comment on a ticket.
type CommentOutput struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id,omitempty"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// SessionOutput represents a work session.
type SessionOutput struct {
	ID            string        `json:"id"`
	StartedAt     time.Time     `json:"started_at"`
	EndedAt       *time.Time    `json:"ended_at,omitempty"`
	Agent         string        `json:"agent"`
	TmuxWindow    string        `json:"tmux_window"`
	CurrentStatus *StatusOutput `json:"current_status,omitempty"`
	IsActive      bool          `json:"is_active"`
}

// StatusOutput represents agent status.
type StatusOutput struct {
	Status string    `json:"status"`
	Tool   *string   `json:"tool,omitempty"`
	Work   *string   `json:"work,omitempty"`
	At     time.Time `json:"at"`
}

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
	Success bool                  `json:"success"`
	Comment CommentOutput         `json:"comment,omitempty"`
	Hooks   *HooksExecutionOutput `json:"hooks,omitempty"`
}

// ConcludeSessionOutput is the output for the concludeSession tool.
type ConcludeSessionOutput struct {
	Success bool                  `json:"success"`
	Message string                `json:"message,omitempty"`
	Hooks   *HooksExecutionOutput `json:"hooks,omitempty"`
}

// SubmitReportOutput is the output for the submitReport tool (deprecated).
type SubmitReportOutput struct {
	Success bool                  `json:"success"`
	Message string                `json:"message,omitempty"`
	Hooks   *HooksExecutionOutput `json:"hooks,omitempty"`
}

// ApproveOutput is the output for the approve tool.
type ApproveOutput struct {
	Success  bool                  `json:"success"`
	TicketID string                `json:"ticket_id"`
	Status   string                `json:"status"`
	Message  string                `json:"message,omitempty"`
	Hooks    *HooksExecutionOutput `json:"hooks,omitempty"`
}

// PickupTicketOutput is the output for the pickupTicket tool.
type PickupTicketOutput struct {
	Success bool                  `json:"success"`
	Message string                `json:"message"`
	Hooks   *HooksExecutionOutput `json:"hooks,omitempty"`
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
		comments[i] = ToCommentOutput(&c)
	}

	return TicketOutput{
		ID:     t.ID,
		Title:  t.Title,
		Body:   t.Body,
		Status: string(status),
		Dates: DatesOutput{
			Created:  t.Dates.Created,
			Updated:  t.Dates.Updated,
			Progress: t.Dates.Progress,
			Reviewed: t.Dates.Reviewed,
			Done:     t.Dates.Done,
		},
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
		currentStatus = &StatusOutput{
			Status: string(s.CurrentStatus.Status),
			Tool:   s.CurrentStatus.Tool,
			Work:   s.CurrentStatus.Work,
			At:     s.CurrentStatus.At,
		}
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

// ToCommentOutput converts a comment to output format.
func ToCommentOutput(c *ticket.Comment) CommentOutput {
	return CommentOutput{
		ID:        c.ID,
		SessionID: c.SessionID,
		Type:      string(c.Type),
		Content:   c.Content,
		CreatedAt: c.CreatedAt,
	}
}
