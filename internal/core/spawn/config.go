package spawn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	daemonconfig "github.com/kareemaly/cortex/internal/daemon/config"
)

// MCPServerConfig represents the MCP server configuration for claude.
type MCPServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// ClaudeMCPConfig represents the claude MCP configuration file format.
type ClaudeMCPConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// MCPConfigParams contains parameters for generating MCP config.
type MCPConfigParams struct {
	CortexdPath string
	TicketID    string // for ticket agents
	TicketType  string // for ticket agents (work/debug/research)
	TicketsDir  string
	ProjectPath string
	TmuxSession string
	DaemonURL   string // optional; defaults to daemonconfig.DefaultDaemonURL
	IsMeta      bool   // if true, pass --meta flag instead of --ticket-id
}

// GenerateMCPConfig creates an MCP configuration for a claude agent.
func GenerateMCPConfig(params MCPConfigParams) *ClaudeMCPConfig {
	args := []string{"mcp"}
	if params.IsMeta {
		args = append(args, "--meta")
	} else if params.TicketID != "" {
		args = append(args, "--ticket-id", params.TicketID)
		if params.TicketType != "" {
			args = append(args, "--ticket-type", params.TicketType)
		}
	}

	serverConfig := MCPServerConfig{
		Command: params.CortexdPath,
		Args:    args,
		Env:     make(map[string]string),
	}

	// Add environment variables
	if params.TicketsDir != "" {
		serverConfig.Env["CORTEX_TICKETS_DIR"] = params.TicketsDir
	}
	if params.ProjectPath != "" {
		serverConfig.Env["CORTEX_PROJECT_PATH"] = params.ProjectPath
	}
	if params.TmuxSession != "" {
		serverConfig.Env["CORTEX_TMUX_SESSION"] = params.TmuxSession
	}

	// Pass daemon URL so sessions can route mutations through the HTTP API
	daemonURL := params.DaemonURL
	if daemonURL == "" {
		daemonURL = daemonconfig.DefaultDaemonURL
	}
	serverConfig.Env["CORTEX_DAEMON_URL"] = daemonURL

	return &ClaudeMCPConfig{
		MCPServers: map[string]MCPServerConfig{
			"cortex": serverConfig,
		},
	}
}

// WriteMCPConfig writes an MCP config to a file.
// Returns the path to the written config file.
func WriteMCPConfig(config *ClaudeMCPConfig, identifier, configDir string) (string, error) {
	if configDir == "" {
		configDir = os.TempDir()
	}

	path := filepath.Join(configDir, fmt.Sprintf("cortex-mcp-%s.json", identifier))

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal MCP config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("write MCP config: %w", err)
	}

	return path, nil
}

// RemoveMCPConfig removes an MCP config file.
func RemoveMCPConfig(path string) error {
	if path == "" {
		return nil
	}
	return os.Remove(path)
}
