package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyEmbeddedDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "claude-code")

	// Copy embedded defaults
	items, err := copyEmbeddedDefaults("claude-code", targetDir, false)
	if err != nil {
		t.Fatalf("copyEmbeddedDefaults failed: %v", err)
	}

	// Verify we got items
	if len(items) == 0 {
		t.Error("expected items to be created, got none")
	}

	// Expected files that should exist
	expectedFiles := []string{
		"cortex.yaml",
		"prompts/architect/SYSTEM.md",
		"prompts/architect/KICKOFF.md",
		"prompts/ticket/work/SYSTEM.md",
		"prompts/ticket/work/KICKOFF.md",
		"prompts/ticket/work/APPROVE.md",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(targetDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", file)
		}
	}

	// Verify cortex.yaml has expected content
	content, err := os.ReadFile(filepath.Join(targetDir, "cortex.yaml"))
	if err != nil {
		t.Fatalf("failed to read cortex.yaml: %v", err)
	}
	if len(content) == 0 {
		t.Error("cortex.yaml is empty")
	}
}

func TestCopyEmbeddedDefaultsNoOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "claude-code")

	// First copy
	_, err := copyEmbeddedDefaults("claude-code", targetDir, false)
	if err != nil {
		t.Fatalf("first copyEmbeddedDefaults failed: %v", err)
	}

	// Modify a file to simulate user customization
	customContent := "# Custom user content\n"
	customPath := filepath.Join(targetDir, "cortex.yaml")
	if err := os.WriteFile(customPath, []byte(customContent), 0644); err != nil {
		t.Fatalf("failed to write custom content: %v", err)
	}

	// Second copy without force - should NOT overwrite
	items, err := copyEmbeddedDefaults("claude-code", targetDir, false)
	if err != nil {
		t.Fatalf("second copyEmbeddedDefaults failed: %v", err)
	}

	// Check that the file was NOT overwritten
	content, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatalf("failed to read cortex.yaml: %v", err)
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
	targetDir := filepath.Join(tmpDir, "claude-code")

	// First copy
	_, err := copyEmbeddedDefaults("claude-code", targetDir, false)
	if err != nil {
		t.Fatalf("first copyEmbeddedDefaults failed: %v", err)
	}

	// Modify a file to simulate user customization
	customContent := "# Custom user content\n"
	customPath := filepath.Join(targetDir, "cortex.yaml")
	if err := os.WriteFile(customPath, []byte(customContent), 0644); err != nil {
		t.Fatalf("failed to write custom content: %v", err)
	}

	// Second copy WITH force - SHOULD overwrite
	_, err = copyEmbeddedDefaults("claude-code", targetDir, true)
	if err != nil {
		t.Fatalf("second copyEmbeddedDefaults with force failed: %v", err)
	}

	// Check that the file WAS overwritten
	content, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatalf("failed to read cortex.yaml: %v", err)
	}
	if string(content) == customContent {
		t.Error("file was NOT overwritten when force=true")
	}
}
