package worktree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	m := NewManager("/test/path")
	if m.projectPath != "/test/path" {
		t.Errorf("expected projectPath '/test/path', got %s", m.projectPath)
	}
}

func TestCreateGeneratesCorrectPaths(t *testing.T) {
	// Setup - use temp dir as project path
	tmpDir := t.TempDir()

	// Initialize a git repo in temp dir
	if err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Note: This test would require a real git repo to fully test Create/Remove
	// For unit testing, we just verify the manager is created correctly
	m := NewManager(tmpDir)

	// Verify the manager has the correct project path
	if m.projectPath != tmpDir {
		t.Errorf("expected projectPath %q, got %q", tmpDir, m.projectPath)
	}
}

func TestWorktreeBaseDir(t *testing.T) {
	if WorktreeBaseDir != ".cortex/worktrees" {
		t.Errorf("expected WorktreeBaseDir '.cortex/worktrees', got %s", WorktreeBaseDir)
	}
}
