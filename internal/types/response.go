package types

import "time"

// ErrorResponse is the standard error response format.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details string `json:"details,omitempty"`
}

// CommentResponse is a comment in a ticket response.
type CommentResponse struct {
	ID      string                 `json:"id"`
	Author  string                 `json:"author"`
	Type    string                 `json:"type"`
	Content string                 `json:"content"`
	Action  *CommentActionResponse `json:"action,omitempty"`
	Created time.Time              `json:"created"`
}

// CommentActionResponse is the action attached to a comment.
type CommentActionResponse struct {
	Type string `json:"type"`
	Args any    `json:"args"`
}

// SessionResponse is a standalone session representation.
type SessionResponse struct {
	TicketID      string    `json:"ticket_id"`
	Agent         string    `json:"agent"`
	TmuxWindow    string    `json:"tmux_window"`
	WorktreePath  *string   `json:"worktree_path,omitempty"`
	FeatureBranch *string   `json:"feature_branch,omitempty"`
	StartedAt     time.Time `json:"started_at"`
	Status        string    `json:"status"`
	Tool          *string   `json:"tool,omitempty"`
}

// TicketResponse is the full ticket response with status.
type TicketResponse struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Title      string            `json:"title"`
	Body       string            `json:"body"`
	Tags       []string          `json:"tags,omitempty"`
	References []string          `json:"references,omitempty"`
	Status     string            `json:"status"`
	Created    time.Time         `json:"created"`
	Updated    time.Time         `json:"updated"`
	Due        *time.Time        `json:"due,omitempty"`
	Comments   []CommentResponse `json:"comments"`
}

// TicketSummary is a brief view of a ticket for lists.
type TicketSummary struct {
	ID               string     `json:"id"`
	Type             string     `json:"type"`
	Title            string     `json:"title"`
	Tags             []string   `json:"tags,omitempty"`
	Status           string     `json:"status"`
	Created          time.Time  `json:"created"`
	Updated          time.Time  `json:"updated"`
	Due              *time.Time `json:"due,omitempty"`
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
	ID         string            `json:"id"`
	Title      string            `json:"title"`
	Category   string            `json:"category"`
	Tags       []string          `json:"tags"`
	References []string          `json:"references"`
	Body       string            `json:"body"`
	Created    string            `json:"created"`
	Updated    string            `json:"updated"`
	Comments   []CommentResponse `json:"comments,omitempty"`
}

// DocSummary is a brief view of a doc for lists.
type DocSummary struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
	Snippet  string   `json:"snippet,omitempty"`
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
	Status      *string    `json:"status,omitempty"`
	Tool        *string    `json:"tool,omitempty"`
	IsOrphaned  bool       `json:"is_orphaned,omitempty"`
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
