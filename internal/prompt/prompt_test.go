package prompt

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPromptsDir(t *testing.T) {
	got := PromptsDir("/project/root")
	want := "/project/root/.cortex/prompts"
	if got != want {
		t.Errorf("PromptsDir() = %q, want %q", got, want)
	}
}

func TestArchitectPath(t *testing.T) {
	got := ArchitectPath("/project/root")
	want := "/project/root/.cortex/prompts/architect.md"
	if got != want {
		t.Errorf("ArchitectPath() = %q, want %q", got, want)
	}
}

func TestTicketAgentPath(t *testing.T) {
	got := TicketAgentPath("/project/root")
	want := "/project/root/.cortex/prompts/ticket-agent.md"
	if got != want {
		t.Errorf("TicketAgentPath() = %q, want %q", got, want)
	}
}

func TestLoadArchitect(t *testing.T) {
	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, ".cortex", "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	template := `Hello {{.ProjectName}}, session {{.TmuxSession}}`
	if err := os.WriteFile(filepath.Join(promptsDir, "architect.md"), []byte(template), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadArchitect(tmpDir, ArchitectVars{
		ProjectName: "myproject",
		TmuxSession: "mysession",
	})
	if err != nil {
		t.Fatalf("LoadArchitect() error = %v", err)
	}

	want := "Hello myproject, session mysession"
	if got != want {
		t.Errorf("LoadArchitect() = %q, want %q", got, want)
	}
}

func TestLoadTicketAgent(t *testing.T) {
	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, ".cortex", "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	template := `Ticket: {{.TicketID}} - {{.Title}}
Slug: {{.Slug}}
Body: {{.Body}}`
	if err := os.WriteFile(filepath.Join(promptsDir, "ticket-agent.md"), []byte(template), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadTicketAgent(tmpDir, TicketVars{
		TicketID: "2026-01-22-fix-bug",
		Title:    "Fix the bug",
		Body:     "This is the description.",
		Slug:     "fix-the-bug",
	})
	if err != nil {
		t.Fatalf("LoadTicketAgent() error = %v", err)
	}

	want := `Ticket: 2026-01-22-fix-bug - Fix the bug
Slug: fix-the-bug
Body: This is the description.`
	if got != want {
		t.Errorf("LoadTicketAgent() = %q, want %q", got, want)
	}
}

func TestLoadArchitect_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := LoadArchitect(tmpDir, ArchitectVars{})
	if err == nil {
		t.Fatal("LoadArchitect() expected error for missing file")
	}

	var notFoundErr *NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("LoadArchitect() error = %T, want *NotFoundError", err)
	}
}

func TestLoadTicketAgent_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := LoadTicketAgent(tmpDir, TicketVars{})
	if err == nil {
		t.Fatal("LoadTicketAgent() expected error for missing file")
	}

	var notFoundErr *NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("LoadTicketAgent() error = %T, want *NotFoundError", err)
	}
}

func TestLoadArchitect_ParseError(t *testing.T) {
	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, ".cortex", "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Invalid template syntax
	badTemplate := `Hello {{.ProjectName`
	if err := os.WriteFile(filepath.Join(promptsDir, "architect.md"), []byte(badTemplate), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadArchitect(tmpDir, ArchitectVars{ProjectName: "test"})
	if err == nil {
		t.Fatal("LoadArchitect() expected error for invalid template")
	}

	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Errorf("LoadArchitect() error = %T, want *ParseError", err)
	}
}

func TestLoadArchitect_RenderError(t *testing.T) {
	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, ".cortex", "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Template that calls a method that doesn't exist
	badTemplate := `Hello {{.InvalidMethod}}`
	if err := os.WriteFile(filepath.Join(promptsDir, "architect.md"), []byte(badTemplate), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadArchitect(tmpDir, ArchitectVars{ProjectName: "test"})
	if err == nil {
		t.Fatal("LoadArchitect() expected error for invalid field")
	}

	var renderErr *RenderError
	if !errors.As(err, &renderErr) {
		t.Errorf("LoadArchitect() error = %T, want *RenderError", err)
	}
}

func TestNotFoundError_Message(t *testing.T) {
	err := &NotFoundError{Path: "/path/to/template.md"}
	got := err.Error()
	if got == "" {
		t.Error("NotFoundError.Error() returned empty string")
	}
	// Check it contains the hint
	want := "cortex install --project"
	if !contains(got, want) {
		t.Errorf("NotFoundError.Error() = %q, should contain %q", got, want)
	}
}

func TestParseError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	err := &ParseError{Path: "/path", Err: inner}
	if errors.Unwrap(err) != inner {
		t.Error("ParseError.Unwrap() did not return inner error")
	}
}

func TestRenderError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	err := &RenderError{Path: "/path", Err: inner}
	if errors.Unwrap(err) != inner {
		t.Error("RenderError.Unwrap() did not return inner error")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
