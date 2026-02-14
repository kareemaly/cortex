package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyEmbeddedDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "main")

	// Copy embedded defaults
	items, err := CopyEmbeddedDefaults("main", targetDir, false)
	if err != nil {
		t.Fatalf("CopyEmbeddedDefaults failed: %v", err)
	}

	// Verify we got items
	if len(items) == 0 {
		t.Error("expected items to be created, got none")
	}

	// Expected files that should exist (prompts only, no cortex.yaml)
	expectedFiles := []string{
		"prompts/meta/SYSTEM.md",
		"prompts/meta/KICKOFF.md",
		"prompts/architect/SYSTEM.md",
		"prompts/architect/KICKOFF.md",
		"prompts/ticket/work/SYSTEM.md",
		"prompts/ticket/work/KICKOFF.md",
		"prompts/ticket/work/APPROVE.md",
		"prompts/ticket/debug/SYSTEM.md",
		"prompts/ticket/debug/KICKOFF.md",
		"prompts/ticket/debug/APPROVE.md",
		"prompts/ticket/research/SYSTEM.md",
		"prompts/ticket/research/KICKOFF.md",
		"prompts/ticket/research/APPROVE.md",
		"prompts/ticket/chore/SYSTEM.md",
		"prompts/ticket/chore/KICKOFF.md",
		"prompts/ticket/chore/APPROVE.md",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(targetDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", file)
		}
	}
}

func TestCopyEmbeddedDefaultsNoOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "main")

	// First copy
	_, err := CopyEmbeddedDefaults("main", targetDir, false)
	if err != nil {
		t.Fatalf("first CopyEmbeddedDefaults failed: %v", err)
	}

	// Modify a prompt file to simulate user customization
	customContent := "# Custom user content\n"
	customPath := filepath.Join(targetDir, "prompts", "architect", "SYSTEM.md")
	if err := os.WriteFile(customPath, []byte(customContent), 0644); err != nil {
		t.Fatalf("failed to write custom content: %v", err)
	}

	// Second copy without force - should NOT overwrite
	items, err := CopyEmbeddedDefaults("main", targetDir, false)
	if err != nil {
		t.Fatalf("second CopyEmbeddedDefaults failed: %v", err)
	}

	// Check that the file was NOT overwritten
	content, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatalf("failed to read prompt file: %v", err)
	}
	if string(content) != customContent {
		t.Error("file was overwritten when force=false")
	}

	// Verify status indicates file exists
	for _, item := range items {
		if item.Path == customPath && item.Status != StatusExists {
			t.Errorf("expected status Exists for %s, got %v", customPath, item.Status)
		}
	}
}

func TestCopyEmbeddedDefaultsForce(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "main")

	// First copy
	_, err := CopyEmbeddedDefaults("main", targetDir, false)
	if err != nil {
		t.Fatalf("first CopyEmbeddedDefaults failed: %v", err)
	}

	// Modify a prompt file to simulate user customization
	customContent := "# Custom user content\n"
	customPath := filepath.Join(targetDir, "prompts", "architect", "SYSTEM.md")
	if err := os.WriteFile(customPath, []byte(customContent), 0644); err != nil {
		t.Fatalf("failed to write custom content: %v", err)
	}

	// Second copy WITH force - SHOULD overwrite
	_, err = CopyEmbeddedDefaults("main", targetDir, true)
	if err != nil {
		t.Fatalf("second CopyEmbeddedDefaults with force failed: %v", err)
	}

	// Check that the file WAS overwritten
	content, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatalf("failed to read prompt file: %v", err)
	}
	if string(content) == customContent {
		t.Error("file was NOT overwritten when force=true")
	}
}
