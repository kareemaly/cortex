package mcp

import (
	"context"
	"os"
	"path/filepath"

	"github.com/kareemaly/cortex1/internal/ticket"
	"github.com/kareemaly/cortex1/pkg/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Config holds configuration for the MCP server.
type Config struct {
	// TicketID restricts the session to a specific ticket (ticket session).
	// If empty, the session is an architect session with full access.
	TicketID string

	// TicketsDir is the directory where tickets are stored.
	// Defaults to ~/.cortex/tickets if empty.
	TicketsDir string
}

// Server is the MCP server for ticket management.
type Server struct {
	mcpServer *mcp.Server
	store     *ticket.Store
	session   *Session
	config    *Config
}

// NewServer creates a new MCP server with the given configuration.
func NewServer(cfg *Config) (*Server, error) {
	if cfg == nil {
		cfg = &Config{}
	}

	// Set default tickets directory
	ticketsDir := cfg.TicketsDir
	if ticketsDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		ticketsDir = filepath.Join(homeDir, ".cortex", "tickets")
	}

	// Create ticket store
	store, err := ticket.NewStore(ticketsDir)
	if err != nil {
		return nil, err
	}

	// Determine session type
	var session *Session
	if cfg.TicketID != "" {
		session = &Session{
			Type:     SessionTypeTicket,
			TicketID: cfg.TicketID,
		}
	} else {
		session = &Session{
			Type: SessionTypeArchitect,
		}
	}

	// Create MCP server
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "cortex-mcp",
		Version: version.Version,
	}, nil)

	s := &Server{
		mcpServer: mcpServer,
		store:     store,
		session:   session,
		config:    cfg,
	}

	// Register tools based on session type
	if session.Type == SessionTypeArchitect {
		s.registerArchitectTools()
	} else {
		s.registerTicketTools()
	}

	return s, nil
}

// Run starts the MCP server using stdio transport.
func (s *Server) Run(ctx context.Context) error {
	return s.mcpServer.Run(ctx, &mcp.StdioTransport{})
}

// Store returns the ticket store for testing purposes.
func (s *Server) Store() *ticket.Store {
	return s.store
}

// Session returns the current session for testing purposes.
func (s *Server) Session() *Session {
	return s.session
}

// IsArchitectSession returns true if this is an architect session.
func (s *Server) IsArchitectSession() bool {
	return s.session.Type == SessionTypeArchitect
}

// IsTicketSession returns true if this is a ticket session.
func (s *Server) IsTicketSession() bool {
	return s.session.Type == SessionTypeTicket
}
