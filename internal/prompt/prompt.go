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
// Deprecated: Use TicketSystemPath and TicketPath instead.
func TicketAgentPath(projectRoot string) string {
	return filepath.Join(PromptsDir(projectRoot), "ticket-agent.md")
}

// TicketSystemPath returns the path to the ticket system prompt file.
// This contains MCP tool instructions and workflow guidance.
func TicketSystemPath(projectRoot string) string {
	return filepath.Join(PromptsDir(projectRoot), "ticket-system.md")
}

// TicketPath returns the path to the ticket content template.
// This template is used for the dynamic ticket content.
func TicketPath(projectRoot string) string {
	return filepath.Join(PromptsDir(projectRoot), "ticket.md")
}

// TicketWorktreePath returns the path to the ticket worktree template.
// This template includes worktree-specific information.
func TicketWorktreePath(projectRoot string) string {
	return filepath.Join(PromptsDir(projectRoot), "ticket-worktree.md")
}

// ApprovePath returns the path to the approve prompt file.
// This contains instructions for the approval workflow.
func ApprovePath(projectRoot string) string {
	return filepath.Join(PromptsDir(projectRoot), "approve.md")
}

// ApproveWorktreePath returns the path to the approve worktree prompt file.
// This contains instructions for approving worktree-based changes.
func ApproveWorktreePath(projectRoot string) string {
	return filepath.Join(PromptsDir(projectRoot), "approve-worktree.md")
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

// LoadPromptFile reads and returns the content of a prompt file.
func LoadPromptFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", &NotFoundError{Path: path}
		}
		return "", err
	}
	return string(content), nil
}
