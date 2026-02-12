package types

import (
	"strings"
	"time"

	"github.com/kareemaly/cortex/internal/docs"
	"github.com/kareemaly/cortex/internal/session"
	"github.com/kareemaly/cortex/internal/ticket"
)

// TmuxChecker allows checking if a tmux window exists.
type TmuxChecker interface {
	WindowExists(session, windowName string) (bool, error)
}

// ToCommentResponse converts a ticket.Comment to CommentResponse.
func ToCommentResponse(c *ticket.Comment) CommentResponse {
	resp := CommentResponse{
		ID:      c.ID,
		Author:  c.Author,
		Type:    string(c.Type),
		Content: c.Content,
		Created: c.Created,
	}
	if c.Action != nil {
		resp.Action = &CommentActionResponse{
			Type: c.Action.Type,
			Args: c.Action.Args,
		}
	}
	return resp
}

// ToSessionResponse converts a session.Session to SessionResponse.
func ToSessionResponse(s *session.Session) SessionResponse {
	return SessionResponse{
		Type:          string(s.Type),
		TicketID:      s.TicketID,
		Agent:         s.Agent,
		TmuxWindow:    s.TmuxWindow,
		WorktreePath:  s.WorktreePath,
		FeatureBranch: s.FeatureBranch,
		StartedAt:     s.StartedAt,
		Status:        string(s.Status),
		Tool:          s.Tool,
	}
}

// ToTicketResponse converts a ticket.Ticket and status to TicketResponse.
func ToTicketResponse(t *ticket.Ticket, status ticket.Status) TicketResponse {
	comments := make([]CommentResponse, len(t.Comments))
	for i, c := range t.Comments {
		comments[i] = ToCommentResponse(&c)
	}

	return TicketResponse{
		ID:         t.ID,
		Type:       t.Type,
		Title:      t.Title,
		Body:       t.Body,
		Tags:       t.Tags,
		References: t.References,
		Status:     string(status),
		Created:    t.Created,
		Updated:    t.Updated,
		Due:        t.Due,
		Comments:   comments,
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
	var comments []CommentResponse
	for _, c := range d.Comments {
		comments = append(comments, ToCommentResponse(&c))
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
		Comments:   comments,
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

// ToDocSummaryWithQuery converts a docs.Doc to DocSummary with a body snippet
// extracted around the first occurrence of query.
func ToDocSummaryWithQuery(d *docs.Doc, query string) DocSummary {
	s := ToDocSummary(d)
	s.Snippet = ExtractSnippet(d.Body, query, 150)
	return s
}

// ExtractSnippet returns a window of maxLen characters from body centered on
// the first case-insensitive occurrence of query. Adds "..." prefix/suffix
// when truncated. Returns empty string if query is empty or not found.
func ExtractSnippet(body, query string, maxLen int) string {
	if query == "" || body == "" {
		return ""
	}

	lowerBody := strings.ToLower(body)
	lowerQuery := strings.ToLower(query)

	idx := strings.Index(lowerBody, lowerQuery)
	if idx < 0 {
		return ""
	}

	if len(body) <= maxLen {
		return body
	}

	// Center the window around the match.
	half := (maxLen - len(query)) / 2
	start := idx - half
	end := start + maxLen

	if start < 0 {
		start = 0
		end = maxLen
	}
	if end > len(body) {
		end = len(body)
		start = max(end-maxLen, 0)
	}

	snippet := body[start:end]

	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(body) {
		snippet = snippet + "..."
	}

	return snippet
}
