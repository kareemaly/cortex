package session

import "time"

// AgentStatus represents an agent's current activity status.
type AgentStatus string

const (
	AgentStatusStarting      AgentStatus = "starting"
	AgentStatusWorking       AgentStatus = "working"
	AgentStatusIdle          AgentStatus = "idle"
	AgentStatusAwaitingInput AgentStatus = "awaiting_input"
	AgentStatusError         AgentStatus = "error"
	AgentStatusEnded         AgentStatus = "ended"
)

// SessionType distinguishes architect sessions from ticket agent sessions.
type SessionType string

const (
	SessionTypeArchitect SessionType = "architect"
	SessionTypeTicket    SessionType = "ticket"
	SessionTypeCollab    SessionType = "collab"
)

// ArchitectSessionKey is the session store key for the architect session.
// This is used as-is (not shortened via storage.ShortID) because the
// architect is a singleton per project.
const ArchitectSessionKey = "architect"

// Session represents an active work session for a ticket.
// Sessions are ephemeral — deleted when ended.
//
// SessionID is a stable UUID minted at creation time. It is the canonical
// routing key for /agent/status updates so collab and architect sessions
// (which don't have a TicketID) can be addressed uniformly.
type Session struct {
	SessionID  string      `json:"session_id"`
	Type       SessionType `json:"type"`
	TicketID   string      `json:"ticket_id,omitempty"`
	CollabID   string      `json:"collab_id,omitempty"`
	Prompt     string      `json:"prompt,omitempty"`
	Agent      string      `json:"agent"`
	TmuxWindow string      `json:"tmux_window"`
	StartedAt  time.Time   `json:"started_at"`
	Status     AgentStatus `json:"status"`
	Tool       *string     `json:"tool,omitempty"`
	Work       *string     `json:"work,omitempty"`
	// AgentSessionID is the agent tool's internal session identifier
	// (Claude Code's --session-id, Codex's rollout-file name). Cortex
	// records it at spawn time so a Resume can re-attach to the existing
	// transcript instead of starting stateless.
	AgentSessionID string `json:"agent_session_id,omitempty"`
}
