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
