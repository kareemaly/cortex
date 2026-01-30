package prompt

import (
	"os"
	"path/filepath"
)

// Stage constants for prompt file naming.
const (
	StageSystem  = "SYSTEM"
	StageKickoff = "KICKOFF"
	StageApprove = "APPROVE"
)

// PromptsDir returns the path to the prompts directory.
func PromptsDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".cortex", "prompts")
}

// ArchitectPromptPath returns the path to an architect prompt file by stage.
// e.g. .cortex/prompts/architect/SYSTEM.md
func ArchitectPromptPath(projectRoot, stage string) string {
	return filepath.Join(PromptsDir(projectRoot), "architect", stage+".md")
}

// TicketPromptPath returns the path to a ticket prompt file by type and stage.
// e.g. .cortex/prompts/ticket/work/KICKOFF.md
func TicketPromptPath(projectRoot, ticketType, stage string) string {
	return filepath.Join(PromptsDir(projectRoot), "ticket", ticketType, stage+".md")
}

// ValidatePromptFile checks that a prompt file exists.
func ValidatePromptFile(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &NotFoundError{SearchPaths: []string{path}}
		}
		return err
	}
	return nil
}

// LoadPromptFile reads and returns the content of a prompt file.
func LoadPromptFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", &NotFoundError{SearchPaths: []string{path}}
		}
		return "", err
	}
	return string(content), nil
}
