package types

import (
	"github.com/kareemaly/cortex/internal/session"
	"github.com/kareemaly/cortex/internal/ticket"
)

type TmuxChecker interface {
	WindowExists(session, windowName string) (bool, error)
}

func ToSessionResponse(s *session.Session) SessionResponse {
	return SessionResponse{
		Type:       string(s.Type),
		TicketID:   s.TicketID,
		CollabID:   s.CollabID,
		Agent:      s.Agent,
		TmuxWindow: s.TmuxWindow,
		StartedAt:  s.StartedAt,
		Status:     string(s.Status),
		Tool:       s.Tool,
	}
}

func ToTicketResponse(t *ticket.Ticket, status ticket.Status, hasConclusion bool) TicketResponse {
	return TicketResponse{
		ID:            t.ID,
		Title:         t.Title,
		Body:          t.Body,
		Repo:          t.Repo,
		Path:          t.Path,
		HasConclusion: hasConclusion,
		References:    t.References,
		Status:        string(status),
		Created:       t.Created,
		Updated:       t.Updated,
		Due:           t.Due,
	}
}

func ToTicketSummary(t *ticket.Ticket, status ticket.Status, sess *session.Session, tmuxSession string, checker TmuxChecker) TicketSummary {
	summary := TicketSummary{
		ID:               t.ID,
		Title:            t.Title,
		Repo:             t.Repo,
		Path:             t.Path,
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
		summary.Agent = sess.Agent
		summary.SessionStartedAt = &sess.StartedAt
	}

	if sess != nil && tmuxSession != "" && checker != nil && sess.TmuxWindow != "" {
		exists, err := checker.WindowExists(tmuxSession, sess.TmuxWindow)
		if err == nil && !exists {
			summary.IsOrphaned = true
		}
	}

	return summary
}
