package api

import (
	"time"

	"github.com/kareemaly/cortex1/internal/ticket"
)

// ErrorResponse is the standard error response format.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details string `json:"details,omitempty"`
}

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

// TicketResponse is the full ticket response with status.
type TicketResponse struct {
	ID       string            `json:"id"`
	Title    string            `json:"title"`
	Body     string            `json:"body"`
	Status   string            `json:"status"`
	Dates    DatesResponse     `json:"dates"`
	Sessions []SessionResponse `json:"sessions"`
}

// DatesResponse is the dates portion of a ticket response.
type DatesResponse struct {
	Created  time.Time  `json:"created"`
	Updated  time.Time  `json:"updated"`
	Approved *time.Time `json:"approved"`
}

// SessionResponse is a session in a ticket response.
type SessionResponse struct {
	ID            string                `json:"id"`
	StartedAt     time.Time             `json:"started_at"`
	EndedAt       *time.Time            `json:"ended_at,omitempty"`
	Agent         string                `json:"agent"`
	TmuxWindow    string                `json:"tmux_window"`
	GitBase       map[string]string     `json:"git_base"`
	Report        ReportResponse        `json:"report"`
	CurrentStatus *StatusEntryResponse  `json:"current_status,omitempty"`
	StatusHistory []StatusEntryResponse `json:"status_history"`
}

// ReportResponse is the report portion of a session response.
type ReportResponse struct {
	Files        []string `json:"files"`
	ScopeChanges *string  `json:"scope_changes,omitempty"`
	Decisions    []string `json:"decisions"`
	Summary      string   `json:"summary"`
}

// StatusEntryResponse is a status entry in a session response.
type StatusEntryResponse struct {
	Status string    `json:"status"`
	Tool   *string   `json:"tool,omitempty"`
	Work   *string   `json:"work,omitempty"`
	At     time.Time `json:"at"`
}

// TicketSummary is a brief view of a ticket for lists.
type TicketSummary struct {
	ID                string    `json:"id"`
	Title             string    `json:"title"`
	Status            string    `json:"status"`
	Created           time.Time `json:"created"`
	HasActiveSessions bool      `json:"has_active_sessions"`
}

// ListAllTicketsResponse groups tickets by status.
type ListAllTicketsResponse struct {
	Backlog  []TicketSummary `json:"backlog"`
	Progress []TicketSummary `json:"progress"`
	Done     []TicketSummary `json:"done"`
}

// ListTicketsResponse is a list of tickets with a single status.
type ListTicketsResponse struct {
	Tickets []TicketSummary `json:"tickets"`
}

// toTicketResponse converts a ticket to its API response form.
func toTicketResponse(t *ticket.Ticket, status ticket.Status) TicketResponse {
	sessions := make([]SessionResponse, len(t.Sessions))
	for i, s := range t.Sessions {
		sessions[i] = toSessionResponse(s)
	}

	return TicketResponse{
		ID:     t.ID,
		Title:  t.Title,
		Body:   t.Body,
		Status: string(status),
		Dates: DatesResponse{
			Created:  t.Dates.Created,
			Updated:  t.Dates.Updated,
			Approved: t.Dates.Approved,
		},
		Sessions: sessions,
	}
}

// toSessionResponse converts a session to its API response form.
func toSessionResponse(s ticket.Session) SessionResponse {
	history := make([]StatusEntryResponse, len(s.StatusHistory))
	for i, h := range s.StatusHistory {
		history[i] = StatusEntryResponse{
			Status: string(h.Status),
			Tool:   h.Tool,
			Work:   h.Work,
			At:     h.At,
		}
	}

	var currentStatus *StatusEntryResponse
	if s.CurrentStatus != nil {
		currentStatus = &StatusEntryResponse{
			Status: string(s.CurrentStatus.Status),
			Tool:   s.CurrentStatus.Tool,
			Work:   s.CurrentStatus.Work,
			At:     s.CurrentStatus.At,
		}
	}

	return SessionResponse{
		ID:         s.ID,
		StartedAt:  s.StartedAt,
		EndedAt:    s.EndedAt,
		Agent:      s.Agent,
		TmuxWindow: s.TmuxWindow,
		GitBase:    s.GitBase,
		Report: ReportResponse{
			Files:        s.Report.Files,
			ScopeChanges: s.Report.ScopeChanges,
			Decisions:    s.Report.Decisions,
			Summary:      s.Report.Summary,
		},
		CurrentStatus: currentStatus,
		StatusHistory: history,
	}
}

// toSummaryList converts a slice of tickets to summaries.
func toSummaryList(tickets []*ticket.Ticket, status ticket.Status) []TicketSummary {
	summaries := make([]TicketSummary, len(tickets))
	for i, t := range tickets {
		summaries[i] = TicketSummary{
			ID:                t.ID,
			Title:             t.Title,
			Status:            string(status),
			Created:           t.Dates.Created,
			HasActiveSessions: t.HasActiveSessions(),
		}
	}
	return summaries
}
