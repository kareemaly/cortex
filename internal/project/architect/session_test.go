package architect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_NoFile(t *testing.T) {
	tmpDir := t.TempDir()

	session, err := Load(tmpDir)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if session != nil {
		t.Error("expected nil session when file doesn't exist")
	}
}

func TestLoad_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	cortexDir := filepath.Join(tmpDir, ".cortex")
	if err := os.MkdirAll(cortexDir, 0755); err != nil {
		t.Fatal(err)
	}

	startedAt := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	data := `{
  "id": "session-123",
  "tmux_session": "cortex",
  "tmux_window": "architect",
  "started_at": "2025-01-15T10:00:00Z"
}`
	if err := os.WriteFile(filepath.Join(cortexDir, stateFileName), []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	session, err := Load(tmpDir)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if session == nil {
		t.Fatal("expected session, got nil")
	}
	if session.ID != "session-123" {
		t.Errorf("expected ID 'session-123', got: %s", session.ID)
	}
	if session.TmuxSession != "cortex" {
		t.Errorf("expected TmuxSession 'cortex', got: %s", session.TmuxSession)
	}
	if session.TmuxWindow != "architect" {
		t.Errorf("expected TmuxWindow 'architect', got: %s", session.TmuxWindow)
	}
	if !session.StartedAt.Equal(startedAt) {
		t.Errorf("expected StartedAt %v, got: %v", startedAt, session.StartedAt)
	}
	if session.EndedAt != nil {
		t.Error("expected EndedAt to be nil")
	}
}

func TestLoad_WithEndedAt(t *testing.T) {
	tmpDir := t.TempDir()
	cortexDir := filepath.Join(tmpDir, ".cortex")
	if err := os.MkdirAll(cortexDir, 0755); err != nil {
		t.Fatal(err)
	}

	endedAt := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	data := `{
  "id": "session-456",
  "tmux_session": "cortex",
  "tmux_window": "architect",
  "started_at": "2025-01-15T10:00:00Z",
  "ended_at": "2025-01-15T12:00:00Z"
}`
	if err := os.WriteFile(filepath.Join(cortexDir, stateFileName), []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	session, err := Load(tmpDir)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if session == nil {
		t.Fatal("expected session, got nil")
	}
	if session.EndedAt == nil {
		t.Fatal("expected EndedAt to be set")
	}
	if !session.EndedAt.Equal(endedAt) {
		t.Errorf("expected EndedAt %v, got: %v", endedAt, session.EndedAt)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	cortexDir := filepath.Join(tmpDir, ".cortex")
	if err := os.MkdirAll(cortexDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(cortexDir, stateFileName), []byte("not valid json"), 0644); err != nil {
		t.Fatal(err)
	}

	session, err := Load(tmpDir)

	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if session != nil {
		t.Error("expected nil session on error")
	}

}

func TestSave_NewSession(t *testing.T) {
	tmpDir := t.TempDir()
	cortexDir := filepath.Join(tmpDir, ".cortex")
	if err := os.MkdirAll(cortexDir, 0755); err != nil {
		t.Fatal(err)
	}

	session := &Session{
		ID:          "session-789",
		TmuxSession: "test-session",
		TmuxWindow:  "test-window",
		StartedAt:   time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	err := Save(tmpDir, session)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify file was written
	statePath := filepath.Join(cortexDir, stateFileName)
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("expected file to exist, got: %v", err)
	}

	var loaded Session
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("expected valid JSON, got: %v", err)
	}
	if loaded.ID != "session-789" {
		t.Errorf("expected ID 'session-789', got: %s", loaded.ID)
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	session := &Session{
		ID:          "session-abc",
		TmuxSession: "test",
		TmuxWindow:  "arch",
		StartedAt:   time.Now(),
	}

	err := Save(tmpDir, session)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify directory and file were created
	statePath := filepath.Join(tmpDir, ".cortex", stateFileName)
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Error("expected file to be created")
	}
}

func TestSave_NilSession(t *testing.T) {
	tmpDir := t.TempDir()

	err := Save(tmpDir, nil)

	if err == nil {
		t.Fatal("expected error for nil session")
	}

	if _, ok := err.(*NilSessionError); !ok {
		t.Errorf("expected NilSessionError, got: %T", err)
	}
}

func TestClear_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	cortexDir := filepath.Join(tmpDir, ".cortex")
	if err := os.MkdirAll(cortexDir, 0755); err != nil {
		t.Fatal(err)
	}

	statePath := filepath.Join(cortexDir, stateFileName)
	if err := os.WriteFile(statePath, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	err := Clear(tmpDir)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Error("expected file to be removed")
	}
}

func TestClear_NoFile(t *testing.T) {
	tmpDir := t.TempDir()

	err := Clear(tmpDir)

	if err != nil {
		t.Fatalf("expected no error when file doesn't exist, got: %v", err)
	}
}

func TestIsActive(t *testing.T) {
	tests := []struct {
		name     string
		session  *Session
		expected bool
	}{
		{
			name: "active session (no end time)",
			session: &Session{
				ID:        "session-1",
				StartedAt: time.Now(),
			},
			expected: true,
		},
		{
			name: "ended session",
			session: func() *Session {
				now := time.Now()
				return &Session{
					ID:        "session-2",
					StartedAt: now.Add(-time.Hour),
					EndedAt:   &now,
				}
			}(),
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.session.IsActive() != tc.expected {
				t.Errorf("expected IsActive() = %v", tc.expected)
			}
		})
	}
}
