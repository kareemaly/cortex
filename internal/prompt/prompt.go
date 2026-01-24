package prompt

import (
	"os"
	"path/filepath"
)

// PromptsDir returns the path to the prompts directory.
func PromptsDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".cortex", "prompts")
}

// ArchitectPath returns the path to the architect prompt file.
func ArchitectPath(projectRoot string) string {
	return filepath.Join(PromptsDir(projectRoot), "architect.md")
}

// TicketAgentPath returns the path to the ticket agent prompt file.
func TicketAgentPath(projectRoot string) string {
	return filepath.Join(PromptsDir(projectRoot), "ticket-agent.md")
}

// ValidatePromptFile checks that a prompt file exists.
func ValidatePromptFile(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &NotFoundError{Path: path}
		}
		return err
	}
	return nil
}
