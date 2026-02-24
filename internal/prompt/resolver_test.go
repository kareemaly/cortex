package prompt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPromptResolver_ResolveArchitectPrompt(t *testing.T) {
	t.Run("finds prompt in project", func(t *testing.T) {
		projectRoot := t.TempDir()
		createPromptFile(t, projectRoot, "architect", "SYSTEM.md", "project system prompt")

		resolver := NewPromptResolver(projectRoot, "")
		content, err := resolver.ResolveArchitectPrompt(StageSystem)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if content != "project system prompt" {
			t.Errorf("expected 'project system prompt', got %q", content)
		}
	})

	t.Run("falls back to base when not in project", func(t *testing.T) {
		projectRoot := t.TempDir()
		baseRoot := t.TempDir()
		createBasePromptFile(t, baseRoot, "architect", "SYSTEM.md", "base system prompt")

		resolver := NewPromptResolver(projectRoot, baseRoot)
		content, err := resolver.ResolveArchitectPrompt(StageSystem)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if content != "base system prompt" {
			t.Errorf("expected 'base system prompt', got %q", content)
		}
	})

	t.Run("project overrides base", func(t *testing.T) {
		projectRoot := t.TempDir()
		baseRoot := t.TempDir()
		createPromptFile(t, projectRoot, "architect", "SYSTEM.md", "project override")
		createBasePromptFile(t, baseRoot, "architect", "SYSTEM.md", "base system prompt")

		resolver := NewPromptResolver(projectRoot, baseRoot)
		content, err := resolver.ResolveArchitectPrompt(StageSystem)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if content != "project override" {
			t.Errorf("expected 'project override', got %q", content)
		}
	})

	t.Run("returns error when not found", func(t *testing.T) {
		projectRoot := t.TempDir()

		resolver := NewPromptResolver(projectRoot, "")
		_, err := resolver.ResolveArchitectPrompt(StageSystem)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		notFoundErr, ok := err.(*NotFoundError)
		if !ok {
			t.Fatalf("expected *NotFoundError, got %T", err)
		}
		if notFoundErr.Role != "architect" {
			t.Errorf("expected role 'architect', got %q", notFoundErr.Role)
		}
		if notFoundErr.Stage != StageSystem {
			t.Errorf("expected stage %q, got %q", StageSystem, notFoundErr.Stage)
		}
	})
}

func TestPromptResolver_ResolveTicketPrompt(t *testing.T) {
	t.Run("finds prompt in project", func(t *testing.T) {
		projectRoot := t.TempDir()
		createTicketPromptFile(t, projectRoot, "work", "KICKOFF.md", "project kickoff")

		resolver := NewPromptResolver(projectRoot, "")
		content, err := resolver.ResolveTicketPrompt("work", StageKickoff)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if content != "project kickoff" {
			t.Errorf("expected 'project kickoff', got %q", content)
		}
	})

	t.Run("falls back to base when not in project", func(t *testing.T) {
		projectRoot := t.TempDir()
		baseRoot := t.TempDir()
		createBaseTicketPromptFile(t, baseRoot, "work", "KICKOFF.md", "base kickoff")

		resolver := NewPromptResolver(projectRoot, baseRoot)
		content, err := resolver.ResolveTicketPrompt("work", StageKickoff)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if content != "base kickoff" {
			t.Errorf("expected 'base kickoff', got %q", content)
		}
	})

	t.Run("falls back to work type when requested type not found", func(t *testing.T) {
		projectRoot := t.TempDir()
		createTicketPromptFile(t, projectRoot, "work", "KICKOFF.md", "work kickoff")

		resolver := NewPromptResolver(projectRoot, "")
		content, err := resolver.ResolveTicketPrompt("research", StageKickoff)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if content != "work kickoff" {
			t.Errorf("expected 'work kickoff', got %q", content)
		}
	})
}

// createPromptFile creates a prompt file for architect prompts in a project root.
func createPromptFile(t *testing.T, root, role, filename, content string) {
	t.Helper()
	dir := filepath.Join(root, "prompts", role)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create prompt dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}
}

// createTicketPromptFile creates a prompt file for ticket prompts in a project root.
func createTicketPromptFile(t *testing.T, root, ticketType, filename, content string) {
	t.Helper()
	dir := filepath.Join(root, "prompts", ticketType)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create prompt dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}
}

// createBasePromptFile creates a prompt file for architect prompts in a base config directory.
func createBasePromptFile(t *testing.T, baseRoot, role, filename, content string) {
	t.Helper()
	dir := filepath.Join(baseRoot, "prompts", role)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create prompt dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}
}

// createBaseTicketPromptFile creates a prompt file for ticket prompts in a base config directory.
func createBaseTicketPromptFile(t *testing.T, baseRoot, ticketType, filename, content string) {
	t.Helper()
	dir := filepath.Join(baseRoot, "prompts", ticketType)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create prompt dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}
}

func TestPromptResolver_ResolveArchitectPromptWithPath(t *testing.T) {
	t.Run("returns correct source path from project", func(t *testing.T) {
		projectRoot := t.TempDir()
		createPromptFile(t, projectRoot, "architect", "SYSTEM.md", "project system prompt")

		resolver := NewPromptResolver(projectRoot, "")
		resolved, err := resolver.ResolveArchitectPromptWithPath(StageSystem)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedPath := filepath.Join(projectRoot, "prompts", "architect", "SYSTEM.md")
		if resolved.SourcePath != expectedPath {
			t.Errorf("expected source path %q, got %q", expectedPath, resolved.SourcePath)
		}
	})

	t.Run("returns correct source path from base", func(t *testing.T) {
		projectRoot := t.TempDir()
		baseRoot := t.TempDir()
		createBasePromptFile(t, baseRoot, "architect", "SYSTEM.md", "base system prompt")

		resolver := NewPromptResolver(projectRoot, baseRoot)
		resolved, err := resolver.ResolveArchitectPromptWithPath(StageSystem)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedPath := filepath.Join(baseRoot, "prompts", "architect", "SYSTEM.md")
		if resolved.SourcePath != expectedPath {
			t.Errorf("expected source path %q, got %q", expectedPath, resolved.SourcePath)
		}
	})

	t.Run("error includes all searched paths", func(t *testing.T) {
		projectRoot := t.TempDir()
		baseRoot := t.TempDir()

		resolver := NewPromptResolver(projectRoot, baseRoot)
		_, err := resolver.ResolveArchitectPromptWithPath(StageSystem)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		notFoundErr, ok := err.(*NotFoundError)
		if !ok {
			t.Fatalf("expected *NotFoundError, got %T", err)
		}
		if len(notFoundErr.SearchPaths) != 2 {
			t.Errorf("expected 2 search paths, got %d", len(notFoundErr.SearchPaths))
		}
		if notFoundErr.Role != "architect" {
			t.Errorf("expected role 'architect', got %q", notFoundErr.Role)
		}
		if notFoundErr.Stage != StageSystem {
			t.Errorf("expected stage %q, got %q", StageSystem, notFoundErr.Stage)
		}
	})

	t.Run("error includes only project path when no base", func(t *testing.T) {
		projectRoot := t.TempDir()

		resolver := NewPromptResolver(projectRoot, "")
		_, err := resolver.ResolveArchitectPromptWithPath(StageSystem)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		notFoundErr, ok := err.(*NotFoundError)
		if !ok {
			t.Fatalf("expected *NotFoundError, got %T", err)
		}
		if len(notFoundErr.SearchPaths) != 1 {
			t.Errorf("expected 1 search path, got %d", len(notFoundErr.SearchPaths))
		}
	})
}

func TestPromptResolver_ResolveTicketPromptWithPath(t *testing.T) {
	t.Run("returns correct source path from project", func(t *testing.T) {
		projectRoot := t.TempDir()
		createTicketPromptFile(t, projectRoot, "work", "KICKOFF.md", "project kickoff")

		resolver := NewPromptResolver(projectRoot, "")
		resolved, err := resolver.ResolveTicketPromptWithPath("work", StageKickoff)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedPath := filepath.Join(projectRoot, "prompts", "work", "KICKOFF.md")
		if resolved.SourcePath != expectedPath {
			t.Errorf("expected source path %q, got %q", expectedPath, resolved.SourcePath)
		}
	})

	t.Run("error includes ticket type", func(t *testing.T) {
		projectRoot := t.TempDir()
		baseRoot := t.TempDir()

		resolver := NewPromptResolver(projectRoot, baseRoot)
		_, err := resolver.ResolveTicketPromptWithPath("work", StageKickoff)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		notFoundErr, ok := err.(*NotFoundError)
		if !ok {
			t.Fatalf("expected *NotFoundError, got %T", err)
		}
		if notFoundErr.TicketType != "work" {
			t.Errorf("expected ticket type 'work', got %q", notFoundErr.TicketType)
		}
	})
}
