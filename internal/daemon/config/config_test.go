package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegisterProject(t *testing.T) {
	cfg := DefaultConfig()
	added := cfg.RegisterProject("/tmp/myproject", "My Project")
	if !added {
		t.Fatal("expected RegisterProject to return true for new project")
	}
	if len(cfg.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(cfg.Projects))
	}
	if cfg.Projects[0].Path != "/tmp/myproject" {
		t.Errorf("expected path /tmp/myproject, got %s", cfg.Projects[0].Path)
	}
	if cfg.Projects[0].Title != "My Project" {
		t.Errorf("expected title My Project, got %s", cfg.Projects[0].Title)
	}
}

func TestRegisterProjectIdempotent(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RegisterProject("/tmp/myproject", "My Project")
	added := cfg.RegisterProject("/tmp/myproject", "Different Title")
	if added {
		t.Fatal("expected RegisterProject to return false for duplicate path")
	}
	if len(cfg.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(cfg.Projects))
	}
}

func TestUnregisterProject(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RegisterProject("/tmp/myproject", "My Project")
	removed := cfg.UnregisterProject("/tmp/myproject")
	if !removed {
		t.Fatal("expected UnregisterProject to return true")
	}
	if len(cfg.Projects) != 0 {
		t.Fatalf("expected 0 projects, got %d", len(cfg.Projects))
	}
}

func TestUnregisterProjectNotFound(t *testing.T) {
	cfg := DefaultConfig()
	removed := cfg.UnregisterProject("/tmp/nonexistent")
	if removed {
		t.Fatal("expected UnregisterProject to return false for missing project")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "settings.yaml")

	cfg := DefaultConfig()
	cfg.Projects = []ProjectEntry{
		{Path: "/home/user/project1", Title: "Project One"},
		{Path: "/home/user/project2", Title: ""},
	}

	if err := cfg.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	loaded, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if len(loaded.Projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(loaded.Projects))
	}
	if loaded.Projects[0].Path != "/home/user/project1" {
		t.Errorf("expected path /home/user/project1, got %s", loaded.Projects[0].Path)
	}
	if loaded.Projects[0].Title != "Project One" {
		t.Errorf("expected title Project One, got %s", loaded.Projects[0].Title)
	}
	if loaded.Projects[1].Path != "/home/user/project2" {
		t.Errorf("expected path /home/user/project2, got %s", loaded.Projects[1].Path)
	}
	if loaded.Port != 4200 {
		t.Errorf("expected default port 4200, got %d", loaded.Port)
	}
}

func TestLoadFromFileMissing(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nonexistent.yaml")

	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if cfg.Port != 4200 {
		t.Errorf("expected default port 4200, got %d", cfg.Port)
	}
	if len(cfg.Projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(cfg.Projects))
	}
}

func TestSaveToFileCreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "settings.yaml")

	cfg := DefaultConfig()
	if err := cfg.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
}

func TestDefaultConfigNotifications(t *testing.T) {
	cfg := DefaultConfig()

	// Channels
	if !cfg.Notifications.Channels.Local.Enabled {
		t.Error("expected channels.local.enabled to be true")
	}
	if !cfg.Notifications.Channels.Local.Sound {
		t.Error("expected channels.local.sound to be true")
	}

	// Behavior
	if cfg.Notifications.Behavior.BatchWindowSeconds != 60 {
		t.Errorf("expected behavior.batch_window_seconds to be 60, got %d", cfg.Notifications.Behavior.BatchWindowSeconds)
	}
	if !cfg.Notifications.Behavior.NotifyOnFirstOnly {
		t.Error("expected behavior.notify_on_first_only to be true")
	}
	if cfg.Notifications.Behavior.ReminderAfterMinutes != 10 {
		t.Errorf("expected behavior.reminder_after_minutes to be 10, got %d", cfg.Notifications.Behavior.ReminderAfterMinutes)
	}
	if !cfg.Notifications.Behavior.SuppressWhenAttached {
		t.Error("expected behavior.suppress_when_attached to be true")
	}

	// Events
	if !cfg.Notifications.Events.AgentWaitingPermission {
		t.Error("expected events.agent_waiting_permission to be true")
	}
	if !cfg.Notifications.Events.AgentIdle {
		t.Error("expected events.agent_idle to be true")
	}
	if !cfg.Notifications.Events.AgentError {
		t.Error("expected events.agent_error to be true")
	}
	if !cfg.Notifications.Events.TicketReviewRequested {
		t.Error("expected events.ticket_review_requested to be true")
	}
}

func TestNotificationsConfigRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "settings.yaml")

	cfg := DefaultConfig()
	// Modify some notification settings
	cfg.Notifications.Channels.Local.Enabled = false
	cfg.Notifications.Channels.Local.Sound = false
	cfg.Notifications.Behavior.BatchWindowSeconds = 120
	cfg.Notifications.Behavior.NotifyOnFirstOnly = false
	cfg.Notifications.Behavior.ReminderAfterMinutes = 5
	cfg.Notifications.Behavior.SuppressWhenAttached = false
	cfg.Notifications.Events.AgentWaitingPermission = false
	cfg.Notifications.Events.AgentIdle = false
	cfg.Notifications.Events.AgentError = false
	cfg.Notifications.Events.TicketReviewRequested = false

	if err := cfg.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	loaded, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Verify channels
	if loaded.Notifications.Channels.Local.Enabled {
		t.Error("expected channels.local.enabled to be false")
	}
	if loaded.Notifications.Channels.Local.Sound {
		t.Error("expected channels.local.sound to be false")
	}

	// Verify behavior
	if loaded.Notifications.Behavior.BatchWindowSeconds != 120 {
		t.Errorf("expected behavior.batch_window_seconds to be 120, got %d", loaded.Notifications.Behavior.BatchWindowSeconds)
	}
	if loaded.Notifications.Behavior.NotifyOnFirstOnly {
		t.Error("expected behavior.notify_on_first_only to be false")
	}
	if loaded.Notifications.Behavior.ReminderAfterMinutes != 5 {
		t.Errorf("expected behavior.reminder_after_minutes to be 5, got %d", loaded.Notifications.Behavior.ReminderAfterMinutes)
	}
	if loaded.Notifications.Behavior.SuppressWhenAttached {
		t.Error("expected behavior.suppress_when_attached to be false")
	}

	// Verify events
	if loaded.Notifications.Events.AgentWaitingPermission {
		t.Error("expected events.agent_waiting_permission to be false")
	}
	if loaded.Notifications.Events.AgentIdle {
		t.Error("expected events.agent_idle to be false")
	}
	if loaded.Notifications.Events.AgentError {
		t.Error("expected events.agent_error to be false")
	}
	if loaded.Notifications.Events.TicketReviewRequested {
		t.Error("expected events.ticket_review_requested to be false")
	}
}

func TestNotificationsPartialYAML(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "settings.yaml")

	// Write partial YAML that only sets some notification values
	partialYAML := `port: 5000
notifications:
  channels:
    local:
      enabled: false
  behavior:
    batch_window_seconds: 30
`
	if err := os.WriteFile(path, []byte(partialYAML), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	loaded, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Verify overridden values
	if loaded.Port != 5000 {
		t.Errorf("expected port 5000, got %d", loaded.Port)
	}
	if loaded.Notifications.Channels.Local.Enabled {
		t.Error("expected channels.local.enabled to be false (overridden)")
	}
	if loaded.Notifications.Behavior.BatchWindowSeconds != 30 {
		t.Errorf("expected behavior.batch_window_seconds to be 30, got %d", loaded.Notifications.Behavior.BatchWindowSeconds)
	}

	// Verify defaults are preserved for unset fields
	if !loaded.Notifications.Channels.Local.Sound {
		t.Error("expected channels.local.sound to use default true")
	}
	if !loaded.Notifications.Behavior.NotifyOnFirstOnly {
		t.Error("expected behavior.notify_on_first_only to use default true")
	}
	if loaded.Notifications.Behavior.ReminderAfterMinutes != 10 {
		t.Errorf("expected behavior.reminder_after_minutes to use default 10, got %d", loaded.Notifications.Behavior.ReminderAfterMinutes)
	}
	if !loaded.Notifications.Behavior.SuppressWhenAttached {
		t.Error("expected behavior.suppress_when_attached to use default true")
	}
	if !loaded.Notifications.Events.AgentWaitingPermission {
		t.Error("expected events.agent_waiting_permission to use default true")
	}
	if !loaded.Notifications.Events.AgentIdle {
		t.Error("expected events.agent_idle to use default true")
	}
	if !loaded.Notifications.Events.AgentError {
		t.Error("expected events.agent_error to use default true")
	}
	if !loaded.Notifications.Events.TicketReviewRequested {
		t.Error("expected events.ticket_review_requested to use default true")
	}
}
