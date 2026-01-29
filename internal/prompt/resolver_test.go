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
		createPromptFile(t, baseRoot, "architect", "SYSTEM.md", "base system prompt")

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
		createPromptFile(t, baseRoot, "architect", "SYSTEM.md", "base system prompt")

		resolver := NewPromptResolver(projectRoot, baseRoot)
		content, err := resolver.ResolveArchitectPrompt(StageSystem)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if content != "project override" {
			t.Errorf("expected 'project override', got %q", content)
		}
	})

	t.Run("partial override - project SYSTEM, base KICKOFF", func(t *testing.T) {
		projectRoot := t.TempDir()
		baseRoot := t.TempDir()
		createPromptFile(t, projectRoot, "architect", "SYSTEM.md", "project system")
		createPromptFile(t, baseRoot, "architect", "KICKOFF.md", "base kickoff")

		resolver := NewPromptResolver(projectRoot, baseRoot)

		// SYSTEM should come from project
		systemContent, err := resolver.ResolveArchitectPrompt(StageSystem)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if systemContent != "project system" {
			t.Errorf("expected 'project system', got %q", systemContent)
		}

		// KICKOFF should come from base
		kickoffContent, err := resolver.ResolveArchitectPrompt(StageKickoff)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if kickoffContent != "base kickoff" {
			t.Errorf("expected 'base kickoff', got %q", kickoffContent)
		}
	})

	t.Run("returns error when not found anywhere", func(t *testing.T) {
		projectRoot := t.TempDir()
		baseRoot := t.TempDir()

		resolver := NewPromptResolver(projectRoot, baseRoot)
		_, err := resolver.ResolveArchitectPrompt(StageSystem)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if _, ok := err.(*NotFoundError); !ok {
			t.Errorf("expected NotFoundError, got %T: %v", err, err)
		}
	})

	t.Run("returns error when no base and not in project", func(t *testing.T) {
		projectRoot := t.TempDir()

		resolver := NewPromptResolver(projectRoot, "")
		_, err := resolver.ResolveArchitectPrompt(StageSystem)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if _, ok := err.(*NotFoundError); !ok {
			t.Errorf("expected NotFoundError, got %T: %v", err, err)
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
		createTicketPromptFile(t, baseRoot, "work", "APPROVE.md", "base approve")

		resolver := NewPromptResolver(projectRoot, baseRoot)
		content, err := resolver.ResolveTicketPrompt("work", StageApprove)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if content != "base approve" {
			t.Errorf("expected 'base approve', got %q", content)
		}
	})

	t.Run("project overrides base for same ticket type", func(t *testing.T) {
		projectRoot := t.TempDir()
		baseRoot := t.TempDir()
		createTicketPromptFile(t, projectRoot, "work", "SYSTEM.md", "project system")
		createTicketPromptFile(t, baseRoot, "work", "SYSTEM.md", "base system")

		resolver := NewPromptResolver(projectRoot, baseRoot)
		content, err := resolver.ResolveTicketPrompt("work", StageSystem)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if content != "project system" {
			t.Errorf("expected 'project system', got %q", content)
		}
	})

	t.Run("different ticket types independent", func(t *testing.T) {
		projectRoot := t.TempDir()
		baseRoot := t.TempDir()
		createTicketPromptFile(t, projectRoot, "work", "SYSTEM.md", "work system")
		createTicketPromptFile(t, baseRoot, "investigation", "SYSTEM.md", "investigation system")

		resolver := NewPromptResolver(projectRoot, baseRoot)

		workContent, err := resolver.ResolveTicketPrompt("work", StageSystem)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if workContent != "work system" {
			t.Errorf("expected 'work system', got %q", workContent)
		}

		invContent, err := resolver.ResolveTicketPrompt("investigation", StageSystem)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if invContent != "investigation system" {
			t.Errorf("expected 'investigation system', got %q", invContent)
		}
	})
}

// createPromptFile creates a prompt file for architect prompts.
func createPromptFile(t *testing.T, root, role, filename, content string) {
	t.Helper()
	dir := filepath.Join(root, ".cortex", "prompts", role)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create prompt dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}
}

// createTicketPromptFile creates a prompt file for ticket prompts.
func createTicketPromptFile(t *testing.T, root, ticketType, filename, content string) {
	t.Helper()
	dir := filepath.Join(root, ".cortex", "prompts", "ticket", ticketType)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create prompt dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}
}
