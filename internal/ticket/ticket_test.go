package ticket

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTicketJSONRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	approved := now.Add(time.Hour)
	ended := now.Add(30 * time.Minute)
	tool := "Edit"
	work := "Writing code"
	scope := "Added extra feature"

	original := Ticket{
		ID:    "test-id",
		Title: "Test Ticket",
		Body:  "Test body content",
		Dates: Dates{
			Created:  now,
			Updated:  now,
			Approved: &approved,
		},
		Sessions: []Session{
			{
				ID:         "session-1",
				StartedAt:  now,
				EndedAt:    &ended,
				Agent:      "claude",
				TmuxWindow: "test-window",
				GitBase:    map[string]string{".": "abc123"},
				Report: Report{
					Files:        []string{"file1.go", "file2.go"},
					ScopeChanges: &scope,
					Decisions:    []string{"Decision 1"},
					Summary:      "Test summary",
				},
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
	if len(decoded.Sessions) != 1 {
		t.Errorf("Sessions count: got %d, want 1", len(decoded.Sessions))
	}
	if decoded.Dates.Approved == nil {
		t.Error("Approved date should not be nil")
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

func TestTicketHasActiveSessions(t *testing.T) {
	now := time.Now()

	ticketWithActive := Ticket{
		Sessions: []Session{
			{EndedAt: &now},
			{EndedAt: nil},
		},
	}
	if !ticketWithActive.HasActiveSessions() {
		t.Error("Ticket with one active session should return true")
	}

	ticketAllEnded := Ticket{
		Sessions: []Session{
			{EndedAt: &now},
			{EndedAt: &now},
		},
	}
	if ticketAllEnded.HasActiveSessions() {
		t.Error("Ticket with all ended sessions should return false")
	}

	ticketNoSessions := Ticket{Sessions: []Session{}}
	if ticketNoSessions.HasActiveSessions() {
		t.Error("Ticket with no sessions should return false")
	}
}

func TestStatusConstants(t *testing.T) {
	if StatusBacklog != "backlog" {
		t.Error("StatusBacklog should be 'backlog'")
	}
	if StatusProgress != "progress" {
		t.Error("StatusProgress should be 'progress'")
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
