package types

import (
	"github.com/kareemaly/cortex/internal/session"
	"github.com/kareemaly/cortex/internal/ticket"
)

// TmuxChecker allows checking if a tmux window exists.
type TmuxChecker interface {
	WindowExists(session, windowName string) (bool, error)
}

// ToSessionResponse converts a session.Session to SessionResponse.
func ToSessionResponse(s *session.Session) SessionResponse {
	return SessionResponse{
		Type:       string(s.Type),
		TicketID:   s.TicketID,
		Agent:      s.Agent,
		TmuxWindow: s.TmuxWindow,
		StartedAt:  s.StartedAt,
		Status:     string(s.Status),
		Tool:       s.Tool,
	}
}

// ToTicketResponse converts a ticket.Ticket and status to TicketResponse.
func ToTicketResponse(t *ticket.Ticket, status ticket.Status) TicketResponse {
	return TicketResponse{
		ID:         t.ID,
		Type:       t.Type,
		Title:      t.Title,
		Body:       t.Body,
		Repo:       t.Repo,
		Session:    t.Session,
		Tags:       t.Tags,
		References: t.References,
		Status:     string(status),
		Created:    t.Created,
		Updated:    t.Updated,
		Due:        t.Due,
	}
}

// ToTicketSummary converts a ticket.Ticket and status to TicketSummary.
// If sess is non-nil, populates session-related fields (HasActiveSession, AgentStatus, AgentTool).
// If tmuxSession and checker are provided, detects orphaned sessions.
func ToTicketSummary(t *ticket.Ticket, status ticket.Status, sess *session.Session, tmuxSession string, checker TmuxChecker) TicketSummary {
	summary := TicketSummary{
		ID:               t.ID,
		Type:             t.Type,
		Title:            t.Title,
		Tags:             t.Tags,
		Status:           string(status),
		Created:          t.Created,
		Updated:          t.Updated,
		Due:              t.Due,
		HasActiveSession: sess != nil,
	}

	if sess != nil {
		statusStr := string(sess.Status)
		summary.AgentStatus = &statusStr
		summary.AgentTool = sess.Tool
		summary.SessionStartedAt = &sess.StartedAt
	}

	// Detect orphaned sessions: active session but tmux window no longer exists.
	if sess != nil && tmuxSession != "" && checker != nil && sess.TmuxWindow != "" {
		exists, err := checker.WindowExists(tmuxSession, sess.TmuxWindow)
		if err == nil && !exists {
			summary.IsOrphaned = true
		}
	}

	return summary
}
