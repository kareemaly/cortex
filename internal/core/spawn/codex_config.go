package spawn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	daemonconfig "github.com/kareemaly/cortex/internal/daemon/config"
)

// WriteCodexConfigDir creates a temp CODEX_HOME directory with config.toml,
// hooks.json, and optionally cortex-system.md. cortexSessionID is embedded as
// a query parameter in every hook URL so the daemon can back-correlate the
// native codex session ID on SessionStart. The caller is responsible for
// adding the returned path to CleanupDirs.
func WriteCodexConfigDir(mcpConfig *ClaudeMCPConfig, systemPrompt string, agentType AgentType, identifier, cortexSessionID string) (string, error) {
	dir, err := os.MkdirTemp("", "cortex-codex-"+identifier+"-*")
	if err != nil {
		return "", fmt.Errorf("create codex config dir: %w", err)
	}

	// Only write system prompt file when content is non-empty.
	// Collab sessions have no system prompt; codex rejects an empty instructions file.
	var systemPromptPath string
	if strings.TrimSpace(systemPrompt) != "" {
		systemPromptPath = filepath.Join(dir, "cortex-system.md")
		if err := os.WriteFile(systemPromptPath, []byte(systemPrompt), 0600); err != nil {
			_ = os.RemoveAll(dir)
			return "", fmt.Errorf("write codex system prompt: %w", err)
		}
	}

	// Build and write config.toml
	toml := buildCodexConfigTOML(mcpConfig, systemPromptPath, agentType)
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(toml), 0600); err != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("write codex config.toml: %w", err)
	}

	// Write a per-session hooks.json so codex fires hooks against the cortexd
	// endpoint. The cortexSessionID is embedded as ?cortex_session_id=<uuid> so
	// the daemon can back-correlate the native codex session ID on SessionStart.
	hooksJSON, err := buildCodexHooksJSON(cortexSessionID)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("write codex hooks.json: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hooks.json"), hooksJSON, 0600); err != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("write codex hooks.json: %w", err)
	}

	// Symlink host auth.json so the user isn't prompted to log in on each spawn.
	// Use a symlink (not a copy) so token refreshes flow back to the real file and
	// credentials never get written to /tmp. Silently skip if the file doesn't exist.
	hostAuth := filepath.Join(os.Getenv("HOME"), ".codex", "auth.json")
	if _, err := os.Stat(hostAuth); err == nil {
		_ = os.Symlink(hostAuth, filepath.Join(dir, "auth.json"))
	}

	return dir, nil
}

// buildCodexConfigTOML generates the TOML content for codex's config.toml.
// Ticket agents use developer_instructions_file (additive, mirrors --append-system-prompt).
// Architect and collab agents use model_instructions_file (full replace, mirrors --system-prompt).
func buildCodexConfigTOML(mcpConfig *ClaudeMCPConfig, systemPromptPath string, agentType AgentType) string {
	var sb strings.Builder

	// codex_hooks must be enabled so the hooks.json we write alongside this
	// config is actually executed. Without this flag codex ignores hooks.json.
	sb.WriteString("[features]\ncodex_hooks = true\n\n")

	if systemPromptPath != "" {
		if agentType == AgentTypeTicketAgent {
			// Additive layer on top of codex's built-in instructions
			fmt.Fprintf(&sb, "developer_instructions_file = %q\n\n", systemPromptPath)
		} else {
			// Full replace of codex's built-in instructions
			fmt.Fprintf(&sb, "model_instructions_file = %q\n\n", systemPromptPath)
		}
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

// buildCodexHooksJSON returns the JSON content for a per-session hooks.json.
// cortexSessionID is appended as ?cortex_session_id=<uuid> so the daemon can
// back-correlate the native codex session ID to the cortex session on
// SessionStart (analogous to the CORTEX_SESSION_ID injection for OpenCode).
func buildCodexHooksJSON(cortexSessionID string) ([]byte, error) {
	hookURL := daemonconfig.DefaultDaemonURL + "/hook/codex"
	if cortexSessionID != "" {
		hookURL += "?cortex_session_id=" + cortexSessionID
	}
	cmd := fmt.Sprintf(
		"curl -s -X POST --max-time 5 -H 'Content-Type: application/json' --data-binary @- %s",
		hookURL,
	)

	type hookEntry struct {
		Type    string `json:"type"`
		Command string `json:"command"`
		Timeout int    `json:"timeout"`
	}
	type hookGroup struct {
		Matcher string      `json:"matcher"`
		Hooks   []hookEntry `json:"hooks"`
	}

	events := []string{"SessionStart", "UserPromptSubmit", "PreToolUse", "PostToolUse", "Stop"}
	hooksMap := make(map[string][]hookGroup, len(events))
	for _, event := range events {
		hooksMap[event] = []hookGroup{{
			Matcher: "",
			Hooks:   []hookEntry{{Type: "command", Command: cmd, Timeout: 10}},
		}}
	}

	return json.MarshalIndent(map[string]any{"hooks": hooksMap}, "", "  ")
}
