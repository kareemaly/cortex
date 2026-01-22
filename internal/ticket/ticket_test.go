package ticket

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTicketJSONRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	progress := now.Add(time.Hour)
	reviewed := now.Add(2 * time.Hour)
	done := now.Add(3 * time.Hour)
	ended := now.Add(30 * time.Minute)
	tool := "Edit"
	work := "Writing code"

	original := Ticket{
		ID:    "test-id",
		Title: "Test Ticket",
		Body:  "Test body content",
		Dates: Dates{
			Created:  now,
			Updated:  now,
			Progress: &progress,
			Reviewed: &reviewed,
			Done:     &done,
		},
		Comments: []Comment{
			{
				ID:        "comment-1",
				SessionID: "session-1",
				Type:      CommentDecision,
				Content:   "Test decision",
				CreatedAt: now,
			},
		},
		Session: &Session{
			ID:              "session-1",
			ClaudeSessionID: "claude-session-123",
			StartedAt:       now,
			EndedAt:         &ended,
			Agent:           "claude",
			TmuxWindow:      "test-window",
			CurrentStatus: &StatusEntry{
				Status: AgentStatusInProgress,
				Tool:   &tool,
				Work:   &work,
				At:     now,
			},
			StatusHistory: []StatusEntry{
				{Status: AgentStatusStarting, At: now},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Ticket
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID mismatch: got %q, want %q", decoded.ID, original.ID)
	}
	if decoded.Title != original.Title {
		t.Errorf("Title mismatch: got %q, want %q", decoded.Title, original.Title)
	}
	if decoded.Session == nil {
		t.Fatal("Session should not be nil")
	}
	if decoded.Session.ClaudeSessionID != "claude-session-123" {
		t.Errorf("ClaudeSessionID mismatch: got %q, want %q", decoded.Session.ClaudeSessionID, "claude-session-123")
	}
	if len(decoded.Comments) != 1 {
		t.Errorf("Comments count: got %d, want 1", len(decoded.Comments))
	}
	if decoded.Dates.Done == nil {
		t.Error("Done date should not be nil")
	}
}

func TestSessionIsActive(t *testing.T) {
	now := time.Now()

	activeSession := Session{EndedAt: nil}
	if !activeSession.IsActive() {
		t.Error("Session with nil EndedAt should be active")
	}

	endedSession := Session{EndedAt: &now}
	if endedSession.IsActive() {
		t.Error("Session with EndedAt set should not be active")
	}
}

func TestTicketHasActiveSession(t *testing.T) {
	now := time.Now()

	ticketWithActive := Ticket{
		Session: &Session{EndedAt: nil},
	}
	if !ticketWithActive.HasActiveSession() {
		t.Error("Ticket with active session should return true")
	}

	ticketEnded := Ticket{
		Session: &Session{EndedAt: &now},
	}
	if ticketEnded.HasActiveSession() {
		t.Error("Ticket with ended session should return false")
	}

	ticketNoSession := Ticket{Session: nil}
	if ticketNoSession.HasActiveSession() {
		t.Error("Ticket with no session should return false")
	}
}

func TestStatusConstants(t *testing.T) {
	if StatusBacklog != "backlog" {
		t.Error("StatusBacklog should be 'backlog'")
	}
	if StatusProgress != "progress" {
		t.Error("StatusProgress should be 'progress'")
	}
	if StatusReview != "review" {
		t.Error("StatusReview should be 'review'")
	}
	if StatusDone != "done" {
		t.Error("StatusDone should be 'done'")
	}
}

func TestAgentStatusConstants(t *testing.T) {
	statuses := []AgentStatus{
		AgentStatusStarting,
		AgentStatusInProgress,
		AgentStatusIdle,
		AgentStatusWaitingPermission,
		AgentStatusError,
	}

	expected := []string{"starting", "in_progress", "idle", "waiting_permission", "error"}

	for i, s := range statuses {
		if string(s) != expected[i] {
			t.Errorf("AgentStatus %d: got %q, want %q", i, s, expected[i])
		}
	}
}
