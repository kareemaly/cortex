package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kareemaly/cortex/internal/daemon/mcp"
	"github.com/spf13/cobra"
)

var (
	mcpTicketID   string
	mcpTicketType string
	mcpMeta       bool
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the MCP server for AI agent integration",
	Long: `Starts the MCP (Model Context Protocol) server for AI agent integration.

The server exposes ticket management tools over stdio transport.

Session types:
  Architect session (default): Full access to all tools
  Ticket session (--ticket-id): Limited to assigned ticket

Examples:
  cortexd mcp                    # Start architect session
  cortexd mcp --ticket-id abc    # Start ticket session for ticket abc`,
	RunE: runMCP,
}

func init() {
	mcpCmd.Flags().StringVar(&mcpTicketID, "ticket-id", "", "Ticket ID for ticket sessions")
	mcpCmd.Flags().StringVar(&mcpTicketType, "ticket-type", "", "Ticket type for ticket sessions (work/debug/research)")
	mcpCmd.Flags().BoolVar(&mcpMeta, "meta", false, "Start a meta session (global, above architects)")
	rootCmd.AddCommand(mcpCmd)
}

func runMCP(cmd *cobra.Command, args []string) error {
	// Get ticket ID from flag only.
	// Do NOT fall back to CORTEX_TICKET_ID env var — that env var is set
	// in the launcher for lifecycle hooks (e.g., "architect" for architect
	// sessions) and must not leak into MCP session type detection.
	// The MCP config explicitly passes --ticket-id for ticket sessions.
	ticketID := mcpTicketID

	// Get optional config from environment
	projectPath := os.Getenv("CORTEX_PROJECT_PATH")
	tmuxSession := os.Getenv("CORTEX_TMUX_SESSION")
	daemonURL := os.Getenv("CORTEX_DAEMON_URL")

	// Create MCP server config
	cfg := &mcp.Config{
		TicketID:    ticketID,
		TicketType:  mcpTicketType,
		IsMeta:      mcpMeta,
		ProjectPath: projectPath,
		TmuxSession: tmuxSession,
		DaemonURL:   daemonURL,
	}

	// Create MCP server
	server, err := mcp.NewServer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create MCP server: %w", err)
	}

	// Create context that cancels on SIGINT or SIGTERM
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	// Run MCP server
	return server.Run(ctx)
}
