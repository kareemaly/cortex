package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegisterArchitect(t *testing.T) {
	cfg := DefaultConfig()
	added := cfg.RegisterArchitect("/tmp/myproject", "My Project")
	if !added {
		t.Fatal("expected RegisterArchitect to return true for new architect")
	}
	if len(cfg.Architects) != 1 {
		t.Fatalf("expected 1 architect, got %d", len(cfg.Architects))
	}
	if cfg.Architects[0].Path != "/tmp/myproject" {
		t.Errorf("expected path /tmp/myproject, got %s", cfg.Architects[0].Path)
	}
	if cfg.Architects[0].Title != "My Project" {
		t.Errorf("expected title My Project, got %s", cfg.Architects[0].Title)
	}
}

func TestRegisterArchitectIdempotent(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RegisterArchitect("/tmp/myproject", "My Project")
	added := cfg.RegisterArchitect("/tmp/myproject", "Different Title")
	if added {
		t.Fatal("expected RegisterArchitect to return false for duplicate path")
	}
	if len(cfg.Architects) != 1 {
		t.Fatalf("expected 1 architect, got %d", len(cfg.Architects))
	}
}

func TestUnregisterArchitect(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RegisterArchitect("/tmp/myproject", "My Project")
	removed := cfg.UnregisterArchitect("/tmp/myproject")
	if !removed {
		t.Fatal("expected UnregisterArchitect to return true")
	}
	if len(cfg.Architects) != 0 {
		t.Fatalf("expected 0 architects, got %d", len(cfg.Architects))
	}
}

func TestUnregisterArchitectNotFound(t *testing.T) {
	cfg := DefaultConfig()
	removed := cfg.UnregisterArchitect("/tmp/nonexistent")
	if removed {
		t.Fatal("expected UnregisterArchitect to return false for missing architect")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "settings.yaml")

	cfg := DefaultConfig()
	cfg.Architects = []ArchitectEntry{
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

	if len(loaded.Architects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(loaded.Architects))
	}
	if loaded.Architects[0].Path != "/home/user/project1" {
		t.Errorf("expected path /home/user/project1, got %s", loaded.Architects[0].Path)
	}
	if loaded.Architects[0].Title != "Project One" {
		t.Errorf("expected title Project One, got %s", loaded.Architects[0].Title)
	}
	if loaded.Architects[1].Path != "/home/user/project2" {
		t.Errorf("expected path /home/user/project2, got %s", loaded.Architects[1].Path)
	}
	if loaded.Port != 4200 {
		t.Errorf("expected default port 4200, got %d", loaded.Port)
	}
	if loaded.BindAddress != "127.0.0.1" {
		t.Errorf("expected default bind_address 127.0.0.1, got %s", loaded.BindAddress)
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
	if len(cfg.Architects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(cfg.Architects))
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
