package spawn

import (
	"fmt"
	"os"
	"path/filepath"
)

// WritePromptFile writes prompt content to a temp file.
// Returns the path to the written file.
func WritePromptFile(content, identifier, suffix, configDir string) (string, error) {
	if configDir == "" {
		configDir = os.TempDir()
	}

	filename := fmt.Sprintf("cortex-%s-%s.txt", suffix, identifier)
	path := filepath.Join(configDir, filename)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write prompt file: %w", err)
	}

	return path, nil
}

// RemovePromptFile removes a prompt temp file.
func RemovePromptFile(path string) error {
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove prompt file: %w", err)
	}
	return nil
}
