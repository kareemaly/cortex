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

func TestArchitectPromptPath(t *testing.T) {
	got := ArchitectPromptPath("/project/root", StageSystem)
	want := "/project/root/.cortex/prompts/architect/SYSTEM.md"
	if got != want {
		t.Errorf("ArchitectPromptPath() = %q, want %q", got, want)
	}
}

func TestTicketPromptPath(t *testing.T) {
	got := TicketPromptPath("/project/root", "work", StageKickoff)
	want := "/project/root/.cortex/prompts/ticket/work/KICKOFF.md"
	if got != want {
		t.Errorf("TicketPromptPath() = %q, want %q", got, want)
	}
}

func TestValidatePromptFile_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, ".cortex", "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	promptFile := filepath.Join(promptsDir, "architect.md")
	if err := os.WriteFile(promptFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	err := ValidatePromptFile(promptFile)
	if err != nil {
		t.Errorf("ValidatePromptFile() error = %v, want nil", err)
	}
}

func TestValidatePromptFile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "nonexistent.md")

	err := ValidatePromptFile(promptFile)
	if err == nil {
		t.Fatal("ValidatePromptFile() expected error for missing file")
	}

	var notFoundErr *NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Errorf("ValidatePromptFile() error = %T, want *NotFoundError", err)
	}
}

func TestNotFoundError_Message(t *testing.T) {
	err := &NotFoundError{Path: "/path/to/prompt.md"}
	got := err.Error()
	if got == "" {
		t.Error("NotFoundError.Error() returned empty string")
	}
	// Check it contains the hint
	want := "cortex init"
	if !contains(got, want) {
		t.Errorf("NotFoundError.Error() = %q, should contain %q", got, want)
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
