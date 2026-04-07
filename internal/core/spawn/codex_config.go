package spawn

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteCodexConfigDir creates a temp CODEX_HOME directory with config.toml and cortex-system.md.
// The caller is responsible for adding the returned path to CleanupDirs.
func WriteCodexConfigDir(mcpConfig *ClaudeMCPConfig, systemPrompt string, agentType AgentType, identifier string) (string, error) {
	dir, err := os.MkdirTemp("", "cortex-codex-"+identifier+"-*")
	if err != nil {
		return "", fmt.Errorf("create codex config dir: %w", err)
	}

	// Write system prompt file (may be empty for resume)
	systemPromptPath := filepath.Join(dir, "cortex-system.md")
	if err := os.WriteFile(systemPromptPath, []byte(systemPrompt), 0600); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("write codex system prompt: %w", err)
	}

	// Build and write config.toml
	toml := buildCodexConfigTOML(mcpConfig, systemPromptPath, agentType)
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(toml), 0600); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("write codex config.toml: %w", err)
	}

	return dir, nil
}

// buildCodexConfigTOML generates the TOML content for codex's config.toml.
// Ticket agents use developer_instructions_file (additive, mirrors --append-system-prompt).
// Architect and collab agents use model_instructions_file (full replace, mirrors --system-prompt).
func buildCodexConfigTOML(mcpConfig *ClaudeMCPConfig, systemPromptPath string, agentType AgentType) string {
	var sb strings.Builder

	if agentType == AgentTypeTicketAgent {
		// Additive layer on top of codex's built-in instructions
		fmt.Fprintf(&sb, "developer_instructions_file = %q\n\n", systemPromptPath)
	} else {
		// Full replace of codex's built-in instructions
		fmt.Fprintf(&sb, "model_instructions_file = %q\n\n", systemPromptPath)
	}

	// MCP servers — codex expects `command` as a string and `args` as an array
	for name, server := range mcpConfig.MCPServers {
		fmt.Fprintf(&sb, "[mcp_servers.%s]\n", name)
		fmt.Fprintf(&sb, "command = %q\n", server.Command)
		if len(server.Args) > 0 {
			fmt.Fprintf(&sb, "args = [%s]\n", tomlStringArray(server.Args))
		} else {
			sb.WriteString("args = []\n")
		}
		if len(server.Env) > 0 {
			fmt.Fprintf(&sb, "\n[mcp_servers.%s.env]\n", name)
			for k, v := range server.Env {
				fmt.Fprintf(&sb, "%s = %q\n", k, v)
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// tomlStringArray formats a slice of strings as a TOML inline array of quoted strings.
func tomlStringArray(items []string) string {
	quoted := make([]string, len(items))
	for i, s := range items {
		quoted[i] = fmt.Sprintf("%q", s)
	}
	return strings.Join(quoted, ", ")
}
