package spawn

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LauncherParams contains all parameters needed to generate a launcher script.
type LauncherParams struct {
	AgentType            string            // agent type: "claude", "opencode", or "copilot"
	PromptFilePath       string            // path to prompt temp file (empty if none)
	SystemPromptFilePath string            // path to system prompt temp file (empty if none)
	ReplaceSystemPrompt  bool              // if true, use --system-prompt (full replace); otherwise --append-system-prompt
	MCPConfigPath        string            // path to MCP config file
	SettingsPath         string            // path to settings config file
	Resume               bool              // if true, emit bare --resume (resume most recent conversation)
	ResumeID             string            // claude session ID to resume (specific conversation)
	SessionID            string            // session ID for --session-id flag
	AgentArgs            []string          // extra CLI args appended to the agent command
	EnvVars              map[string]string // env vars to export (e.g., CORTEX_TICKET_ID)
	CleanupFiles         []string          // temp paths to rm on exit (launcher path is added automatically)
}

// WriteLauncherScript generates and writes a bash launcher script.
// The script uses $(cat file) to read prompts at execution time, avoiding
// inline prompt embedding in tmux send-keys commands.
// Returns the path to the launcher script.
func WriteLauncherScript(params LauncherParams, identifier, configDir string) (string, error) {
	if configDir == "" {
		configDir = os.TempDir()
	}

	filename := fmt.Sprintf("cortex-launcher-%s.sh", identifier)
	path := filepath.Join(configDir, filename)

	// Include the launcher script itself in the cleanup list
	allCleanupFiles := make([]string, len(params.CleanupFiles), len(params.CleanupFiles)+1)
	copy(allCleanupFiles, params.CleanupFiles)
	allCleanupFiles = append(allCleanupFiles, path)

	script := buildLauncherScript(params, allCleanupFiles)

	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		return "", fmt.Errorf("write launcher script: %w", err)
	}

	return path, nil
}

// RemoveLauncherScript removes a launcher script file.
func RemoveLauncherScript(path string) error {
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove launcher script: %w", err)
	}
	return nil
}

// buildLauncherScript generates the bash script content.
// cleanupFiles includes all temp files plus the launcher script itself.
func buildLauncherScript(params LauncherParams, cleanupFiles []string) string {
	var sb strings.Builder

	sb.WriteString("#!/usr/bin/env bash\n")

	// Trap to clean up all temp files on exit
	if len(cleanupFiles) > 0 {
		sb.WriteString("trap 'rm -f")
		for _, f := range cleanupFiles {
			sb.WriteString(" ")
			sb.WriteString(shellQuote(f))
		}
		sb.WriteString("' EXIT\n")
	}

	// Export environment variables
	for k, v := range params.EnvVars {
		sb.WriteString(fmt.Sprintf("export %s=%s\n", k, shellQuote(v)))
	}

	// Build agent command based on type
	var command string
	switch params.AgentType {
	case "copilot":
		command = buildCopilotCommand(params)
	default:
		// Default to claude (works for both "claude" and "opencode")
		command = buildClaudeCommand(params)
	}

	sb.WriteString(command)
	sb.WriteString("\n")

	return sb.String()
}

// buildClaudeCommand builds the claude CLI command string.
func buildClaudeCommand(params LauncherParams) string {
	var parts []string
	parts = append(parts, "claude")

	// Add prompt via $(cat file)
	if params.PromptFilePath != "" {
		parts = append(parts, fmt.Sprintf("\"$(cat %s)\"", shellQuote(params.PromptFilePath)))
	}

	// Add system prompt via $(cat file)
	if params.SystemPromptFilePath != "" {
		flag := "--append-system-prompt"
		if params.ReplaceSystemPrompt {
			flag = "--system-prompt"
		}
		parts = append(parts, flag, fmt.Sprintf("\"$(cat %s)\"", shellQuote(params.SystemPromptFilePath)))
	}

	// Add MCP config
	if params.MCPConfigPath != "" {
		parts = append(parts, "--mcp-config", shellQuote(params.MCPConfigPath))
	}

	// Add settings config
	if params.SettingsPath != "" {
		parts = append(parts, "--settings", shellQuote(params.SettingsPath))
	}

	// Add resume flag
	if params.ResumeID != "" {
		parts = append(parts, "--resume", params.ResumeID)
	} else if params.Resume {
		parts = append(parts, "--resume")
	}

	// Add session ID flag
	if params.SessionID != "" {
		parts = append(parts, "--session-id", params.SessionID)
	}

	// Add extra agent args
	for _, arg := range params.AgentArgs {
		parts = append(parts, shellQuote(arg))
	}

	return strings.Join(parts, " ")
}

// buildCopilotCommand builds the copilot CLI command string.
// Copilot uses different flags than Claude:
//   - Command: `gh copilot agent` instead of `claude`
//   - MCP: `--additional-mcp-config` instead of `--mcp-config`
//   - No `--system-prompt` or `--settings` flags (Copilot doesn't support these)
//   - `--yolo` for automation mode (required)
//   - `--no-custom-instructions` to ignore AGENTS.md files
func buildCopilotCommand(params LauncherParams) string {
	var parts []string
	parts = append(parts, "gh", "copilot", "agent")

	// Required: enable automation mode
	parts = append(parts, "--yolo")

	// Ignore AGENTS.md files (Cortex manages its own prompts)
	parts = append(parts, "--no-custom-instructions")

	// Add prompt via $(cat file)
	if params.PromptFilePath != "" {
		parts = append(parts, fmt.Sprintf("\"$(cat %s)\"", shellQuote(params.PromptFilePath)))
	}

	// Add MCP config (Copilot uses --additional-mcp-config)
	if params.MCPConfigPath != "" {
		parts = append(parts, "--additional-mcp-config", shellQuote(params.MCPConfigPath))
	}

	// Add resume flag (same format as Claude)
	if params.ResumeID != "" {
		parts = append(parts, "--resume", params.ResumeID)
	} else if params.Resume {
		parts = append(parts, "--resume")
	}

	// Note: Copilot doesn't support --session-id, --system-prompt, or --settings
	// SystemPromptFilePath and SettingsPath are ignored for Copilot

	// Add extra agent args
	for _, arg := range params.AgentArgs {
		parts = append(parts, shellQuote(arg))
	}

	return strings.Join(parts, " ")
}

// shellQuote wraps a string in single quotes for safe shell inclusion.
// Single quotes inside the string are escaped using the '\” idiom.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
