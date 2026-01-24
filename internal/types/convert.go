package types

import "github.com/kareemaly/cortex/internal/ticket"

// ToDatesResponse converts ticket.Dates to DatesResponse.
func ToDatesResponse(d ticket.Dates) DatesResponse {
	return DatesResponse{
		Created:  d.Created,
		Updated:  d.Updated,
		Progress: d.Progress,
		Reviewed: d.Reviewed,
		Done:     d.Done,
	}
}

// ToCommentResponse converts a ticket.Comment to CommentResponse.
func ToCommentResponse(c *ticket.Comment) CommentResponse {
	return CommentResponse{
		ID:        c.ID,
		SessionID: c.SessionID,
		Type:      string(c.Type),
		Content:   c.Content,
		CreatedAt: c.CreatedAt,
	}
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

// ToRequestedReviewResponse converts a ticket.ReviewRequest to RequestedReviewResponse.
func ToRequestedReviewResponse(r ticket.ReviewRequest) RequestedReviewResponse {
	return RequestedReviewResponse{
		RepoPath:    r.RepoPath,
		Summary:     r.Summary,
		RequestedAt: r.RequestedAt,
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

	reviews := make([]RequestedReviewResponse, len(s.RequestedReviews))
	for i, r := range s.RequestedReviews {
		reviews[i] = ToRequestedReviewResponse(r)
	}

	return SessionResponse{
		ID:               s.ID,
		StartedAt:        s.StartedAt,
		EndedAt:          s.EndedAt,
		Agent:            s.Agent,
		TmuxWindow:       s.TmuxWindow,
		CurrentStatus:    currentStatus,
		StatusHistory:    history,
		RequestedReviews: reviews,
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
		ID:       t.ID,
		Title:    t.Title,
		Body:     t.Body,
		Status:   string(status),
		Dates:    ToDatesResponse(t.Dates),
		Comments: comments,
		Session:  session,
	}
}

// ToTicketSummary converts a ticket.Ticket and status to TicketSummary.
// If includeAgentStatus is true, populates AgentStatus and AgentTool from active session.
func ToTicketSummary(t *ticket.Ticket, status ticket.Status, includeAgentStatus bool) TicketSummary {
	summary := TicketSummary{
		ID:               t.ID,
		Title:            t.Title,
		Status:           string(status),
		Created:          t.Dates.Created,
		Updated:          t.Dates.Updated,
		HasActiveSession: t.HasActiveSession(),
	}

	if includeAgentStatus && t.HasActiveSession() && t.Session.CurrentStatus != nil {
		statusStr := string(t.Session.CurrentStatus.Status)
		summary.AgentStatus = &statusStr
		summary.AgentTool = t.Session.CurrentStatus.Tool
	}

	return summary
}
