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
	mcpTicketID string
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
	mcpCmd.Flags().StringVar(&mcpTicketID, "ticket-id", "", "Ticket ID for ticket sessions (also reads CORTEX_TICKET_ID env)")
	rootCmd.AddCommand(mcpCmd)
}

func runMCP(cmd *cobra.Command, args []string) error {
	// Get ticket ID from flag or environment
	ticketID := mcpTicketID
	if ticketID == "" {
		ticketID = os.Getenv("CORTEX_TICKET_ID")
	}

	// Get optional config from environment
	ticketsDir := os.Getenv("CORTEX_TICKETS_DIR")
	projectPath := os.Getenv("CORTEX_PROJECT_PATH")
	tmuxSession := os.Getenv("CORTEX_TMUX_SESSION")

	// Create MCP server config
	cfg := &mcp.Config{
		TicketID:    ticketID,
		TicketsDir:  ticketsDir,
		ProjectPath: projectPath,
		TmuxSession: tmuxSession,
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
