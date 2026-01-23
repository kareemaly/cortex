package api

import (
	"strings"
	"time"

	"github.com/kareemaly/cortex/internal/ticket"
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
	Comments []CommentResponse `json:"comments"`
	Session  *SessionResponse  `json:"session,omitempty"`
}

// DatesResponse is the dates portion of a ticket response.
type DatesResponse struct {
	Created  time.Time  `json:"created"`
	Updated  time.Time  `json:"updated"`
	Progress *time.Time `json:"progress,omitempty"`
	Reviewed *time.Time `json:"reviewed,omitempty"`
	Done     *time.Time `json:"done,omitempty"`
}

// CommentResponse is a comment in a ticket response.
type CommentResponse struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id,omitempty"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// SessionResponse is a session in a ticket response.
type SessionResponse struct {
	ID            string                `json:"id"`
	StartedAt     time.Time             `json:"started_at"`
	EndedAt       *time.Time            `json:"ended_at,omitempty"`
	Agent         string                `json:"agent"`
	TmuxWindow    string                `json:"tmux_window"`
	CurrentStatus *StatusEntryResponse  `json:"current_status,omitempty"`
	StatusHistory []StatusEntryResponse `json:"status_history"`
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
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Status           string    `json:"status"`
	Created          time.Time `json:"created"`
	HasActiveSession bool      `json:"has_active_session"`
}

// ListAllTicketsResponse groups tickets by status.
type ListAllTicketsResponse struct {
	Backlog  []TicketSummary `json:"backlog"`
	Progress []TicketSummary `json:"progress"`
	Review   []TicketSummary `json:"review"`
	Done     []TicketSummary `json:"done"`
}

// ListTicketsResponse is a list of tickets with a single status.
type ListTicketsResponse struct {
	Tickets []TicketSummary `json:"tickets"`
}

// SpawnResponse is the response for the spawn endpoint.
type SpawnResponse struct {
	Session SessionResponse `json:"session"`
	Ticket  TicketResponse  `json:"ticket"`
}

// ArchitectStateResponse is the response for GET /architect.
type ArchitectStateResponse struct {
	State   string                    `json:"state"`
	Session *ArchitectSessionResponse `json:"session,omitempty"`
}

// ArchitectSessionResponse is the session details in an architect response.
type ArchitectSessionResponse struct {
	ID          string     `json:"id"`
	TmuxSession string     `json:"tmux_session"`
	TmuxWindow  string     `json:"tmux_window"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
}

// ArchitectSpawnResponse is the response for POST /architect/spawn.
type ArchitectSpawnResponse struct {
	State       string                   `json:"state"`
	Session     ArchitectSessionResponse `json:"session"`
	TmuxSession string                   `json:"tmux_session"`
	TmuxWindow  string                   `json:"tmux_window"`
}

// toTicketResponse converts a ticket to its API response form.
func toTicketResponse(t *ticket.Ticket, status ticket.Status) TicketResponse {
	var session *SessionResponse
	if t.Session != nil {
		s := toSessionResponse(*t.Session)
		session = &s
	}

	comments := make([]CommentResponse, len(t.Comments))
	for i, c := range t.Comments {
		comments[i] = CommentResponse{
			ID:        c.ID,
			SessionID: c.SessionID,
			Type:      string(c.Type),
			Content:   c.Content,
			CreatedAt: c.CreatedAt,
		}
	}

	return TicketResponse{
		ID:     t.ID,
		Title:  t.Title,
		Body:   t.Body,
		Status: string(status),
		Dates: DatesResponse{
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
		ID:            s.ID,
		StartedAt:     s.StartedAt,
		EndedAt:       s.EndedAt,
		Agent:         s.Agent,
		TmuxWindow:    s.TmuxWindow,
		CurrentStatus: currentStatus,
		StatusHistory: history,
	}
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
		summaries = append(summaries, TicketSummary{
			ID:               t.ID,
			Title:            t.Title,
			Status:           string(status),
			Created:          t.Dates.Created,
			HasActiveSession: t.HasActiveSession(),
		})
	}
	if summaries == nil {
		summaries = []TicketSummary{}
	}
	return summaries
}
