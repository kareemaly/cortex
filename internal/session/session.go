package session

import "time"

// AgentStatus represents an agent's current activity status.
type AgentStatus string

const (
	AgentStatusStarting          AgentStatus = "starting"
	AgentStatusInProgress        AgentStatus = "in_progress"
	AgentStatusIdle              AgentStatus = "idle"
	AgentStatusWaitingPermission AgentStatus = "waiting_permission"
	AgentStatusError             AgentStatus = "error"
)

// SessionType distinguishes architect sessions from ticket agent sessions.
type SessionType string

const (
	SessionTypeArchitect SessionType = "architect"
	SessionTypeTicket    SessionType = "ticket"
	SessionTypeMeta      SessionType = "meta"
)

// ArchitectSessionKey is the session store key for the architect session.
// This is used as-is (not shortened via storage.ShortID) because the
// architect is a singleton per project.
const ArchitectSessionKey = "architect"

// MetaSessionKey is the session store key for the meta session.
// The meta agent is a global singleton (one per daemon).
const MetaSessionKey = "meta"

// Session represents an active work session for a ticket.
// Sessions are ephemeral — deleted when ended.
type Session struct {
	Type          SessionType `json:"type"`
	TicketID      string      `json:"ticket_id,omitempty"`
	Agent         string      `json:"agent"`
	TmuxWindow    string      `json:"tmux_window"`
	WorktreePath  *string     `json:"worktree_path,omitempty"`
	FeatureBranch *string     `json:"feature_branch,omitempty"`
	StartedAt     time.Time   `json:"started_at"`
	Status        AgentStatus `json:"status"`
	Tool          *string     `json:"tool,omitempty"`
	Work          *string     `json:"work,omitempty"`
}
