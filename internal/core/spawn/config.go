package spawn

import (
	"github.com/hiveryn/agentruntime"
	daemonconfig "github.com/kareemaly/cortex/internal/daemon/config"
)

// MCPConfigParams contains parameters for generating MCP server config.
type MCPConfigParams struct {
	CortexdPath   string
	TicketID      string
	TicketType    string
	TicketsDir    string
	ArchitectPath string
	TmuxSession   string
	DaemonURL     string // optional; defaults to daemonconfig.DefaultDaemonURL
	StartedAt     string // RFC3339 timestamp of when the session started
	CollabID      string
}

// BuildMCPServerConfig converts Cortex MCP params to an agentruntime
// MCP server config consumed by all adapters.
func BuildMCPServerConfig(params MCPConfigParams) agentruntime.MCPServerConfig {
	args := []string{"mcp"}
	if params.TicketID != "" {
		args = append(args, "--ticket-id", params.TicketID)
		if params.TicketType != "" {
			args = append(args, "--ticket-type", params.TicketType)
		}
	}

	env := make(map[string]string)
	if params.TicketsDir != "" {
		env["CORTEX_TICKETS_DIR"] = params.TicketsDir
	}
	if params.ArchitectPath != "" {
		env["CORTEX_ARCHITECT_PATH"] = params.ArchitectPath
	}
	if params.TmuxSession != "" {
		env["CORTEX_TMUX_SESSION"] = params.TmuxSession
	}

	daemonURL := params.DaemonURL
	if daemonURL == "" {
		daemonURL = daemonconfig.DefaultDaemonURL
	}
	env["CORTEX_DAEMON_URL"] = daemonURL

	if params.StartedAt != "" {
		env["CORTEX_STARTED_AT"] = params.StartedAt
	}
	if params.CollabID != "" {
		env["CORTEX_COLLAB_ID"] = params.CollabID
	}

	return agentruntime.MCPServerConfig{
		Name:    "cortex",
		Command: params.CortexdPath,
		Args:    args,
		Env:     env,
	}
}
