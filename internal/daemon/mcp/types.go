package mcp

import (
	"time"

	"github.com/kareemaly/cortex1/internal/ticket"
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
	Status string `json:"status,omitempty" jsonschema:"Filter by status (backlog/progress/done). Leave empty for all tickets."`
}

// SearchTicketsInput is the input for the searchTickets tool.
type SearchTicketsInput struct {
	Query    string `json:"query" jsonschema:"Search term to match against title and body"`
	FromDate string `json:"from_date,omitempty" jsonschema:"Filter tickets created on or after this date (RFC3339 format)"`
	ToDate   string `json:"to_date,omitempty" jsonschema:"Filter tickets created on or before this date (RFC3339 format)"`
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
	Status string `json:"status" jsonschema:"Target status (backlog/progress/done)"`
}

// SpawnSessionInput is the input for the spawnSession tool.
type SpawnSessionInput struct {
	TicketID string `json:"ticket_id" jsonschema:"The ticket ID to spawn a session for"`
	Agent    string `json:"agent,omitempty" jsonschema:"Agent name (default: claude)"`
}

// GetSessionStatusInput is the input for the getSessionStatus tool.
type GetSessionStatusInput struct {
	TicketID  string `json:"ticket_id" jsonschema:"The ticket ID to get session status for"`
	SessionID string `json:"session_id,omitempty" jsonschema:"Specific session ID (optional, defaults to active session)"`
}

// Input types for ticket tools

// SubmitReportInput is the input for the submitReport tool.
type SubmitReportInput struct {
	Files        []string `json:"files,omitempty" jsonschema:"List of modified files"`
	ScopeChanges *string  `json:"scope_changes,omitempty" jsonschema:"Description of any scope changes"`
	Decisions    []string `json:"decisions,omitempty" jsonschema:"List of decisions made"`
	Summary      string   `json:"summary,omitempty" jsonschema:"Summary of work done"`
}

// ApproveInput is the input for the approve tool.
type ApproveInput struct {
	Summary string `json:"summary,omitempty" jsonschema:"Final summary before approval"`
}

// Output types

// TicketSummary is a brief ticket representation for list views.
type TicketSummary struct {
	ID                string    `json:"id"`
	Title             string    `json:"title"`
	Status            string    `json:"status"`
	Created           time.Time `json:"created"`
	HasActiveSessions bool      `json:"has_active_sessions"`
}

// TicketOutput is the full ticket representation.
type TicketOutput struct {
	ID       string          `json:"id"`
	Title    string          `json:"title"`
	Body     string          `json:"body"`
	Status   string          `json:"status"`
	Dates    DatesOutput     `json:"dates"`
	Sessions []SessionOutput `json:"sessions"`
}

// DatesOutput represents ticket date information.
type DatesOutput struct {
	Created  time.Time  `json:"created"`
	Updated  time.Time  `json:"updated"`
	Approved *time.Time `json:"approved,omitempty"`
}

// SessionOutput represents a work session.
type SessionOutput struct {
	ID            string            `json:"id"`
	StartedAt     time.Time         `json:"started_at"`
	EndedAt       *time.Time        `json:"ended_at,omitempty"`
	Agent         string            `json:"agent"`
	TmuxWindow    string            `json:"tmux_window"`
	GitBase       map[string]string `json:"git_base"`
	Report        ReportOutput      `json:"report"`
	CurrentStatus *StatusOutput     `json:"current_status,omitempty"`
	IsActive      bool              `json:"is_active"`
}

// ReportOutput represents a session report.
type ReportOutput struct {
	Files        []string `json:"files"`
	ScopeChanges *string  `json:"scope_changes,omitempty"`
	Decisions    []string `json:"decisions"`
	Summary      string   `json:"summary"`
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
	Message string `json:"message"`
}

// GetSessionStatusOutput is the output for the getSessionStatus tool.
type GetSessionStatusOutput struct {
	Session *SessionOutput `json:"session,omitempty"`
	Message string         `json:"message,omitempty"`
}

// SubmitReportOutput is the output for the submitReport tool.
type SubmitReportOutput struct {
	Success bool         `json:"success"`
	Report  ReportOutput `json:"report"`
}

// ApproveOutput is the output for the approve tool.
type ApproveOutput struct {
	Success  bool   `json:"success"`
	TicketID string `json:"ticket_id"`
	Status   string `json:"status"`
}

// PickupTicketOutput is the output for the pickupTicket tool.
type PickupTicketOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Conversion functions

// ToTicketOutput converts a ticket and status to output format.
func ToTicketOutput(t *ticket.Ticket, status ticket.Status) TicketOutput {
	sessions := make([]SessionOutput, len(t.Sessions))
	for i, s := range t.Sessions {
		sessions[i] = ToSessionOutput(&s)
	}

	return TicketOutput{
		ID:     t.ID,
		Title:  t.Title,
		Body:   t.Body,
		Status: string(status),
		Dates: DatesOutput{
			Created:  t.Dates.Created,
			Updated:  t.Dates.Updated,
			Approved: t.Dates.Approved,
		},
		Sessions: sessions,
	}
}

// ToTicketSummary converts a ticket and status to summary format.
func ToTicketSummary(t *ticket.Ticket, status ticket.Status) TicketSummary {
	return TicketSummary{
		ID:                t.ID,
		Title:             t.Title,
		Status:            string(status),
		Created:           t.Dates.Created,
		HasActiveSessions: t.HasActiveSessions(),
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
		ID:         s.ID,
		StartedAt:  s.StartedAt,
		EndedAt:    s.EndedAt,
		Agent:      s.Agent,
		TmuxWindow: s.TmuxWindow,
		GitBase:    s.GitBase,
		Report: ReportOutput{
			Files:        s.Report.Files,
			ScopeChanges: s.Report.ScopeChanges,
			Decisions:    s.Report.Decisions,
			Summary:      s.Report.Summary,
		},
		CurrentStatus: currentStatus,
		IsActive:      s.IsActive(),
	}
}
