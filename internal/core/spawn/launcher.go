package spawn

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LauncherParams contains all parameters needed to generate a launcher script.
type LauncherParams struct {
	PromptFilePath       string            // path to prompt temp file (empty if none)
	SystemPromptFilePath string            // path to system prompt temp file (empty if none)
	MCPConfigPath        string            // path to MCP config file
	SettingsPath         string            // path to settings config file
	PermissionMode       string            // "plan" or empty
	AllowedTools         []string          // tools that don't require approval
	ResumeID             string            // claude session ID to resume
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

	// Build claude command
	var parts []string
	parts = append(parts, "claude")

	// Add prompt via $(cat file)
	if params.PromptFilePath != "" {
		parts = append(parts, fmt.Sprintf("\"$(cat %s)\"", shellQuote(params.PromptFilePath)))
	}

	// Add system prompt via $(cat file)
	if params.SystemPromptFilePath != "" {
		parts = append(parts, "--append-system-prompt", fmt.Sprintf("\"$(cat %s)\"", shellQuote(params.SystemPromptFilePath)))
	}

	// Add MCP config
	if params.MCPConfigPath != "" {
		parts = append(parts, "--mcp-config", shellQuote(params.MCPConfigPath))
	}

	// Add settings config
	if params.SettingsPath != "" {
		parts = append(parts, "--settings", shellQuote(params.SettingsPath))
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

	// Add extra agent args
	for _, arg := range params.AgentArgs {
		parts = append(parts, shellQuote(arg))
	}

	sb.WriteString(strings.Join(parts, " "))
	sb.WriteString("\n")

	return sb.String()
}

// shellQuote wraps a string in single quotes for safe shell inclusion.
// Single quotes inside the string are escaped using the '\” idiom.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
