package types

import (
	"time"

	"github.com/kareemaly/cortex/internal/docs"
	"github.com/kareemaly/cortex/internal/ticket"
)

// TmuxChecker allows checking if a tmux window exists.
type TmuxChecker interface {
	WindowExists(session, windowName string) (bool, error)
}

// ToDatesResponse converts ticket.Dates to DatesResponse.
func ToDatesResponse(d ticket.Dates) DatesResponse {
	return DatesResponse{
		Created:  d.Created,
		Updated:  d.Updated,
		Progress: d.Progress,
		Reviewed: d.Reviewed,
		Done:     d.Done,
		DueDate:  d.DueDate,
	}
}

// ToCommentResponse converts a ticket.Comment to CommentResponse.
func ToCommentResponse(c *ticket.Comment) CommentResponse {
	resp := CommentResponse{
		ID:        c.ID,
		SessionID: c.SessionID,
		Type:      string(c.Type),
		Content:   c.Content,
		CreatedAt: c.CreatedAt,
	}
	if c.Action != nil {
		resp.Action = &CommentActionResponse{
			Type: c.Action.Type,
			Args: c.Action.Args,
		}
	}
	return resp
}

// ToStatusEntryResponse converts a ticket.StatusEntry to StatusEntryResponse.
func ToStatusEntryResponse(s ticket.StatusEntry) StatusEntryResponse {
	return StatusEntryResponse{
		Status: string(s.Status),
		Tool:   s.Tool,
		Work:   s.Work,
		At:     s.At,
	}
}

// ToSessionResponse converts a ticket.Session to SessionResponse.
func ToSessionResponse(s ticket.Session) SessionResponse {
	history := make([]StatusEntryResponse, len(s.StatusHistory))
	for i, h := range s.StatusHistory {
		history[i] = ToStatusEntryResponse(h)
	}

	var currentStatus *StatusEntryResponse
	if s.CurrentStatus != nil {
		cs := ToStatusEntryResponse(*s.CurrentStatus)
		currentStatus = &cs
	}

	return SessionResponse{
		ID:            s.ID,
		StartedAt:     s.StartedAt,
		EndedAt:       s.EndedAt,
		Agent:         s.Agent,
		TmuxWindow:    s.TmuxWindow,
		CurrentStatus: currentStatus,
		StatusHistory: history,
	}
}

// ToTicketResponse converts a ticket.Ticket and status to TicketResponse.
func ToTicketResponse(t *ticket.Ticket, status ticket.Status) TicketResponse {
	var session *SessionResponse
	if t.Session != nil {
		s := ToSessionResponse(*t.Session)
		session = &s
	}

	comments := make([]CommentResponse, len(t.Comments))
	for i, c := range t.Comments {
		comments[i] = ToCommentResponse(&c)
	}

	return TicketResponse{
		ID:         t.ID,
		Type:       t.Type,
		Title:      t.Title,
		Body:       t.Body,
		References: t.References,
		Status:     string(status),
		Dates:      ToDatesResponse(t.Dates),
		Comments:   comments,
		Session:    session,
	}
}

// ToTicketSummary converts a ticket.Ticket and status to TicketSummary.
// If includeAgentStatus is true, populates AgentStatus and AgentTool from active session.
// If tmuxSession and checker are provided, detects orphaned sessions (active session but no tmux window).
func ToTicketSummary(t *ticket.Ticket, status ticket.Status, includeAgentStatus bool, tmuxSession string, checker TmuxChecker) TicketSummary {
	summary := TicketSummary{
		ID:               t.ID,
		Type:             t.Type,
		Title:            t.Title,
		Status:           string(status),
		Created:          t.Dates.Created,
		Updated:          t.Dates.Updated,
		DueDate:          t.Dates.DueDate,
		HasActiveSession: t.HasActiveSession(),
	}

	if includeAgentStatus && t.HasActiveSession() && t.Session.CurrentStatus != nil {
		statusStr := string(t.Session.CurrentStatus.Status)
		summary.AgentStatus = &statusStr
		summary.AgentTool = t.Session.CurrentStatus.Tool
	}

	// Detect orphaned sessions: active session but tmux window no longer exists.
	if t.HasActiveSession() && tmuxSession != "" && checker != nil && t.Session.TmuxWindow != "" {
		exists, err := checker.WindowExists(tmuxSession, t.Session.TmuxWindow)
		if err == nil && !exists {
			summary.IsOrphaned = true
		}
	}

	return summary
}

// ToDocResponse converts a docs.Doc to DocResponse.
func ToDocResponse(d *docs.Doc) DocResponse {
	tags := d.Tags
	if tags == nil {
		tags = []string{}
	}
	refs := d.References
	if refs == nil {
		refs = []string{}
	}
	return DocResponse{
		ID:         d.ID,
		Title:      d.Title,
		Category:   d.Category,
		Tags:       tags,
		References: refs,
		Body:       d.Body,
		Created:    d.Created.Format(time.RFC3339),
		Updated:    d.Updated.Format(time.RFC3339),
	}
}

// ToDocSummary converts a docs.Doc to DocSummary.
func ToDocSummary(d *docs.Doc) DocSummary {
	tags := d.Tags
	if tags == nil {
		tags = []string{}
	}
	return DocSummary{
		ID:       d.ID,
		Title:    d.Title,
		Category: d.Category,
		Tags:     tags,
		Created:  d.Created.Format(time.RFC3339),
		Updated:  d.Updated.Format(time.RFC3339),
	}
}
