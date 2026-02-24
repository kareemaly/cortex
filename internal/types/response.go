package types

import "time"

// ErrorResponse is the standard error response format.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details string `json:"details,omitempty"`
}

// SessionResponse is a standalone session representation.
type SessionResponse struct {
	Type       string    `json:"type"`
	TicketID   string    `json:"ticket_id,omitempty"`
	Agent      string    `json:"agent"`
	TmuxWindow string    `json:"tmux_window"`
	StartedAt  time.Time `json:"started_at"`
	Status     string    `json:"status"`
	Tool       *string   `json:"tool,omitempty"`
}

// TicketResponse is the full ticket response with status.
type TicketResponse struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	Title      string     `json:"title"`
	Body       string     `json:"body"`
	Repo       string     `json:"repo,omitempty"`
	Path       string     `json:"path,omitempty"`
	Session    string     `json:"session,omitempty"`
	References []string   `json:"references,omitempty"`
	Status     string     `json:"status"`
	Created    time.Time  `json:"created"`
	Updated    time.Time  `json:"updated"`
	Due        *time.Time `json:"due,omitempty"`
}

// TicketSummary is a brief view of a ticket for lists.
type TicketSummary struct {
	ID               string     `json:"id"`
	Type             string     `json:"type"`
	Title            string     `json:"title"`
	Status           string     `json:"status"`
	Created          time.Time  `json:"created"`
	Updated          time.Time  `json:"updated"`
	Due              *time.Time `json:"due,omitempty"`
	HasActiveSession bool       `json:"has_active_session"`
	AgentStatus      *string    `json:"agent_status,omitempty"`
	AgentTool        *string    `json:"agent_tool,omitempty"`
	IsOrphaned       bool       `json:"is_orphaned,omitempty"`
	SessionStartedAt *time.Time `json:"session_started_at,omitempty"`
}

// ListTicketsResponse is a list of tickets with a single status.
type ListTicketsResponse struct {
	Tickets []TicketSummary `json:"tickets"`
}

// ListAllTicketsResponse groups tickets by status.
type ListAllTicketsResponse struct {
	Backlog  []TicketSummary `json:"backlog"`
	Progress []TicketSummary `json:"progress"`
	Done     []TicketSummary `json:"done"`
}

// ConclusionResponse is the full conclusion response.
type ConclusionResponse struct {
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	Ticket  string    `json:"ticket,omitempty"`
	Repo    string    `json:"repo,omitempty"`
	Body    string    `json:"body"`
	Created time.Time `json:"created"`
}

// ConclusionSummary is metadata-only (no body) for list responses.
type ConclusionSummary struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Ticket      string    `json:"ticket,omitempty"`
	TicketTitle string    `json:"ticket_title,omitempty"`
	Repo        string    `json:"repo,omitempty"`
	Created     time.Time `json:"created"`
}

// ListConclusionsResponse is a paginated list of conclusions (metadata only).
type ListConclusionsResponse struct {
	Conclusions []ConclusionSummary `json:"conclusions"`
	Total       int                 `json:"total"`
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

// HealthResponse is the response structure for the health endpoint.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// ArchitectTicketCounts holds ticket counts by status.
type ArchitectTicketCounts struct {
	Backlog  int `json:"backlog"`
	Progress int `json:"progress"`
	Done     int `json:"done"`
}

// ArchitectResponse represents a single architect in the API response.
type ArchitectResponse struct {
	Path   string                 `json:"path"`
	Title  string                 `json:"title"`
	Exists bool                   `json:"exists"`
	Counts *ArchitectTicketCounts `json:"counts,omitempty"`
}

// ConcludeSessionResponse is the response for concluding a session.
type ConcludeSessionResponse struct {
	Success  bool   `json:"success"`
	TicketID string `json:"ticket_id"`
	Message  string `json:"message"`
}

// ResolvePromptResponse is the response for the resolve prompt endpoint.
type ResolvePromptResponse struct {
	Content    string `json:"content"`
	SourcePath string `json:"source_path"`
}

// PromptFileInfo describes a single prompt file with its ejection status.
type PromptFileInfo struct {
	Path     string `json:"path"`
	Group    string `json:"group"`
	Subgroup string `json:"subgroup,omitempty"`
	Stage    string `json:"stage"`
	Ejected  bool   `json:"ejected"`
	Content  string `json:"content"`
}

// PromptGroupInfo groups related prompt files under a display name.
type PromptGroupInfo struct {
	Name  string           `json:"name"`
	Key   string           `json:"key"`
	Files []PromptFileInfo `json:"files"`
}

// ListPromptsResponse is the response for GET /prompts.
type ListPromptsResponse struct {
	Groups        []PromptGroupInfo `json:"groups"`
	ConfigPath    string            `json:"config_path"`
	ConfigContent string            `json:"config_content"`
}
