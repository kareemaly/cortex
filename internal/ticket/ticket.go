package ticket

import "time"

// Status represents a ticket's workflow status.
type Status string

const (
	StatusBacklog  Status = "backlog"
	StatusProgress Status = "progress"
	StatusReview   Status = "review"
	StatusDone     Status = "done"
)

// CommentType represents the type of comment on a ticket.
type CommentType string

const (
	CommentReviewRequested CommentType = "review_requested"
	CommentDone            CommentType = "done"
	CommentBlocker         CommentType = "blocker"
	CommentGeneral         CommentType = "comment"
)

// Comment represents a comment on a ticket.
type Comment struct {
	ID        string         `json:"id"`
	SessionID string         `json:"session_id,omitempty"`
	Type      CommentType    `json:"type"`
	Content   string         `json:"content"`
	Action    *CommentAction `json:"action,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// CommentAction holds structured data for actionable comments.
type CommentAction struct {
	Type string `json:"type"`
	Args any    `json:"args"`
}

// GitDiffArgs holds the arguments for a git_diff action.
type GitDiffArgs struct {
	RepoPath string `json:"repo_path"`
	Commit   string `json:"commit,omitempty"`
}

// AgentStatus represents an agent's current activity status.
type AgentStatus string

const (
	AgentStatusStarting          AgentStatus = "starting"
	AgentStatusInProgress        AgentStatus = "in_progress"
	AgentStatusIdle              AgentStatus = "idle"
	AgentStatusWaitingPermission AgentStatus = "waiting_permission"
	AgentStatusError             AgentStatus = "error"
)

// DefaultTicketType is the default type for tickets when none is specified.
const DefaultTicketType = "work"

// Ticket represents a work item with sessions and metadata.
type Ticket struct {
	ID       string    `json:"id"`
	Type     string    `json:"type"`
	Title    string    `json:"title"`
	Body     string    `json:"body"`
	Dates    Dates     `json:"dates"`
	Comments []Comment `json:"comments"`
	Session  *Session  `json:"session,omitempty"`
}

// Dates holds the timestamp metadata for a ticket.
type Dates struct {
	Created  time.Time  `json:"created"`
	Updated  time.Time  `json:"updated"`
	Progress *time.Time `json:"progress,omitempty"`
	Reviewed *time.Time `json:"reviewed,omitempty"`
	Done     *time.Time `json:"done,omitempty"`
	DueDate  *time.Time `json:"due_date,omitempty"`
}

// Session represents a work session on a ticket.
type Session struct {
	ID            string        `json:"id"`
	StartedAt     time.Time     `json:"started_at"`
	EndedAt       *time.Time    `json:"ended_at,omitempty"`
	Agent         string        `json:"agent"`
	TmuxWindow    string        `json:"tmux_window"`
	WorktreePath  *string       `json:"worktree_path,omitempty"`
	FeatureBranch *string       `json:"feature_branch,omitempty"`
	CurrentStatus *StatusEntry  `json:"current_status,omitempty"`
	StatusHistory []StatusEntry `json:"status_history"`
}

// StatusEntry represents a point-in-time status of an agent.
type StatusEntry struct {
	Status AgentStatus `json:"status"`
	Tool   *string     `json:"tool"`
	Work   *string     `json:"work"`
	At     time.Time   `json:"at"`
}

// IsActive returns true if the session has not ended.
func (s *Session) IsActive() bool {
	return s.EndedAt == nil
}

// HasActiveSession returns true if the ticket has an active session.
func (t *Ticket) HasActiveSession() bool {
	return t.Session != nil && t.Session.IsActive()
}
