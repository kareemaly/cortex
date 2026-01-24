package spawn

import (
	"strings"
)

// ClaudeCommandParams contains parameters for building a claude command.
type ClaudeCommandParams struct {
	Prompt             string
	AppendSystemPrompt string // system prompt content to pass via --append-system-prompt
	MCPConfigPath      string
	SettingsPath       string   // path to settings.json with hooks config
	PermissionMode     string   // "plan" or "full" or empty (default)
	AllowedTools       []string // tools that don't require user approval
	ResumeID           string   // optional claude session ID to resume
	SessionID          string   // optional session ID for --session-id flag
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
	parts = append(parts, "'"+escapedPrompt+"'")

	// Add append system prompt (static instructions content)
	if params.AppendSystemPrompt != "" {
		escapedSystemPrompt := EscapePromptForShell(params.AppendSystemPrompt)
		parts = append(parts, "--append-system-prompt", "'"+escapedSystemPrompt+"'")
	}

	// Add MCP config
	if params.MCPConfigPath != "" {
		parts = append(parts, "--mcp-config", params.MCPConfigPath)
	}

	// Add settings config (hooks)
	if params.SettingsPath != "" {
		parts = append(parts, "--settings", params.SettingsPath)
	}

	// Add permission mode
	if params.PermissionMode != "" {
		parts = append(parts, "--permission-mode", params.PermissionMode)
	}

	// Add allowed tools
	if len(params.AllowedTools) > 0 {
		parts = append(parts, "--allowedTools", strings.Join(params.AllowedTools, ","))
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

// GenerateWindowName creates a tmux window name from a ticket.
func GenerateWindowName(ticketTitle string) string {
	return generateSlug(ticketTitle)
}

// generateSlug creates a URL-safe slug from a title.
// Moved inline to avoid circular dependency with ticket package.
func generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Keep only alphanumeric and hyphens
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	slug = result.String()
	// Collapse multiple hyphens
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	// Trim leading/trailing hyphens
	slug = strings.Trim(slug, "-")
	// Limit length
	if len(slug) > 50 {
		slug = slug[:50]
	}
	return slug
}
