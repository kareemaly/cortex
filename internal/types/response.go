package types

import "time"

// ErrorResponse is the standard error response format.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details string `json:"details,omitempty"`
}

// DatesResponse is the dates portion of a ticket response.
type DatesResponse struct {
	Created  time.Time  `json:"created"`
	Updated  time.Time  `json:"updated"`
	Progress *time.Time `json:"progress,omitempty"`
	Reviewed *time.Time `json:"reviewed,omitempty"`
	Done     *time.Time `json:"done,omitempty"`
	DueDate  *time.Time `json:"due_date,omitempty"`
}

// CommentResponse is a comment in a ticket response.
type CommentResponse struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id,omitempty"`
	Type      string                 `json:"type"`
	Content   string                 `json:"content"`
	Action    *CommentActionResponse `json:"action,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// CommentActionResponse is the action attached to a comment.
type CommentActionResponse struct {
	Type string `json:"type"`
	Args any    `json:"args"`
}

// StatusEntryResponse is a status entry in a session response.
type StatusEntryResponse struct {
	Status string    `json:"status"`
	Tool   *string   `json:"tool,omitempty"`
	Work   *string   `json:"work,omitempty"`
	At     time.Time `json:"at"`
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

// TicketResponse is the full ticket response with status.
type TicketResponse struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Title      string            `json:"title"`
	Body       string            `json:"body"`
	References []string          `json:"references,omitempty"`
	Status     string            `json:"status"`
	Dates      DatesResponse     `json:"dates"`
	Comments   []CommentResponse `json:"comments"`
	Session    *SessionResponse  `json:"session,omitempty"`
}

// TicketSummary is a brief view of a ticket for lists.
type TicketSummary struct {
	ID               string     `json:"id"`
	Type             string     `json:"type"`
	Title            string     `json:"title"`
	Status           string     `json:"status"`
	Created          time.Time  `json:"created"`
	Updated          time.Time  `json:"updated"`
	DueDate          *time.Time `json:"due_date,omitempty"`
	HasActiveSession bool       `json:"has_active_session"`
	AgentStatus      *string    `json:"agent_status,omitempty"`
	AgentTool        *string    `json:"agent_tool,omitempty"`
	IsOrphaned       bool       `json:"is_orphaned,omitempty"`
}

// ListTicketsResponse is a list of tickets with a single status.
type ListTicketsResponse struct {
	Tickets []TicketSummary `json:"tickets"`
}

// ListAllTicketsResponse groups tickets by status.
type ListAllTicketsResponse struct {
	Backlog  []TicketSummary `json:"backlog"`
	Progress []TicketSummary `json:"progress"`
	Review   []TicketSummary `json:"review"`
	Done     []TicketSummary `json:"done"`
}

// DocResponse is the full doc response.
type DocResponse struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Category   string   `json:"category"`
	Tags       []string `json:"tags"`
	References []string `json:"references"`
	Body       string   `json:"body"`
	Created    string   `json:"created"`
	Updated    string   `json:"updated"`
}

// DocSummary is a brief view of a doc for lists.
type DocSummary struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
	Created  string   `json:"created"`
	Updated  string   `json:"updated"`
}

// ListDocsResponse is a list of docs.
type ListDocsResponse struct {
	Docs []DocSummary `json:"docs"`
}

// ArchitectSessionResponse is the session details in an architect response.
type ArchitectSessionResponse struct {
	ID          string     `json:"id"`
	TmuxSession string     `json:"tmux_session"`
	TmuxWindow  string     `json:"tmux_window"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
}

// ArchitectStateResponse is the response for GET /architect.
type ArchitectStateResponse struct {
	State   string                    `json:"state"`
	Session *ArchitectSessionResponse `json:"session,omitempty"`
}

// ArchitectSpawnResponse is the response for POST /architect/spawn.
type ArchitectSpawnResponse struct {
	State       string                   `json:"state"`
	Session     ArchitectSessionResponse `json:"session"`
	TmuxSession string                   `json:"tmux_session"`
	TmuxWindow  string                   `json:"tmux_window"`
}
