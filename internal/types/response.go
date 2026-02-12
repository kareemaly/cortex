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
	Type          string    `json:"type"`
	TicketID      string    `json:"ticket_id,omitempty"`
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

// HealthResponse is the response structure for the health endpoint.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// ProjectTicketCounts holds ticket counts by status.
type ProjectTicketCounts struct {
	Backlog  int `json:"backlog"`
	Progress int `json:"progress"`
	Review   int `json:"review"`
	Done     int `json:"done"`
}

// ProjectResponse represents a single project in the API response.
type ProjectResponse struct {
	Path   string               `json:"path"`
	Title  string               `json:"title"`
	Exists bool                 `json:"exists"`
	Counts *ProjectTicketCounts `json:"counts,omitempty"`
}

// AddCommentResponse is the response for adding a comment.
type AddCommentResponse struct {
	Success bool            `json:"success"`
	Comment CommentResponse `json:"comment"`
}

// RequestReviewResponse is the response for requesting a review.
type RequestReviewResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Comment CommentResponse `json:"comment"`
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

// TagCount represents a tag and how many times it appears across tickets and docs.
type TagCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// ListTagsResponse is the response for GET /tags.
type ListTagsResponse struct {
	Tags []TagCount `json:"tags"`
}

// MetaSpawnResponse is the response for POST /meta/spawn.
type MetaSpawnResponse struct {
	State       string `json:"state"`
	TmuxSession string `json:"tmux_session"`
	TmuxWindow  string `json:"tmux_window"`
}

// MetaStateResponse is the response for GET /meta.
type MetaStateResponse struct {
	State string `json:"state"`
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
