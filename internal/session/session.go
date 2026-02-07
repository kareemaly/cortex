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

// ArchitectSessionKey is the session store key for the architect session.
// This is used as-is (not shortened via storage.ShortID) because the
// architect is a singleton per project.
const ArchitectSessionKey = "architect"

// Session represents an active work session for a ticket.
// Sessions are ephemeral — deleted when ended.
type Session struct {
	TicketID      string      `json:"ticket_id"`
	Agent         string      `json:"agent"`
	TmuxWindow    string      `json:"tmux_window"`
	WorktreePath  *string     `json:"worktree_path,omitempty"`
	FeatureBranch *string     `json:"feature_branch,omitempty"`
	StartedAt     time.Time   `json:"started_at"`
	Status        AgentStatus `json:"status"`
	Tool          *string     `json:"tool,omitempty"`
}
