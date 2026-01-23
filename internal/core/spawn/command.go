package spawn

import (
	"fmt"
	"strings"

	"github.com/kareemaly/cortex/internal/prompt"
	"github.com/kareemaly/cortex/internal/ticket"
)

// ClaudeCommandParams contains parameters for building a claude command.
type ClaudeCommandParams struct {
	Prompt         string
	MCPConfigPath  string
	PermissionMode string // "plan" or "full" or empty (default)
	ResumeID       string // optional claude session ID to resume
	SessionID      string // optional session ID for --session-id flag
}

// EscapePromptForShell escapes a prompt for safe shell inclusion.
// Uses POSIX single-quote escaping: ' -> '\”
func EscapePromptForShell(prompt string) string {
	return strings.ReplaceAll(prompt, "'", "'\\''")
}

// BuildClaudeCommand constructs the claude CLI command string.
func BuildClaudeCommand(params ClaudeCommandParams) string {
	escapedPrompt := EscapePromptForShell(params.Prompt)

	var parts []string
	parts = append(parts, "claude")

	// Add prompt
	parts = append(parts, fmt.Sprintf("'%s'", escapedPrompt))

	// Add MCP config
	if params.MCPConfigPath != "" {
		parts = append(parts, "--mcp-config", params.MCPConfigPath)
	}

	// Add permission mode
	if params.PermissionMode != "" {
		parts = append(parts, "--permission-mode", params.PermissionMode)
	}

	// Add resume flag
	if params.ResumeID != "" {
		parts = append(parts, "--resume", params.ResumeID)
	}

	// Add session ID flag
	if params.SessionID != "" {
		parts = append(parts, "--session-id", params.SessionID)
	}

	return strings.Join(parts, " ")
}

// TicketPromptVars contains variables for ticket agent prompts.
type TicketPromptVars struct {
	TicketID string
	Title    string
	Body     string
	Slug     string
}

// ArchitectPromptVars contains variables for architect prompts.
type ArchitectPromptVars struct {
	ProjectName string
	TmuxSession string
}

// BuildTicketAgentPrompt loads and renders the ticket agent prompt.
func BuildTicketAgentPrompt(projectPath string, vars TicketPromptVars) (string, error) {
	return prompt.LoadTicketAgent(projectPath, prompt.TicketVars{
		TicketID: vars.TicketID,
		Title:    vars.Title,
		Body:     vars.Body,
		Slug:     vars.Slug,
	})
}

// BuildArchitectPrompt loads and renders the architect prompt.
func BuildArchitectPrompt(projectPath string, vars ArchitectPromptVars) (string, error) {
	return prompt.LoadArchitect(projectPath, prompt.ArchitectVars{
		ProjectName: vars.ProjectName,
		TmuxSession: vars.TmuxSession,
	})
}

// GenerateWindowName creates a tmux window name from a ticket.
func GenerateWindowName(ticketTitle string) string {
	return ticket.GenerateSlug(ticketTitle)
}
