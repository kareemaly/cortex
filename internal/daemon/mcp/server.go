package mcp

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/kareemaly/cortex1/internal/lifecycle"
	"github.com/kareemaly/cortex1/internal/project/config"
	"github.com/kareemaly/cortex1/internal/ticket"
	"github.com/kareemaly/cortex1/internal/tmux"
	"github.com/kareemaly/cortex1/pkg/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Config holds configuration for the MCP server.
type Config struct {
	// TicketID restricts the session to a specific ticket (ticket session).
	// If empty, the session is an architect session with full access.
	TicketID string

	// TicketsDir is the directory where tickets are stored.
	// If empty, derived from ProjectPath/.cortex/tickets.
	TicketsDir string

	// ProjectPath is the project root for hook execution.
	// If set, project config is loaded from this path.
	// Required if TicketsDir is not set.
	ProjectPath string

	// TmuxSession is the tmux session name for spawning agents.
	// Required for spawn operations - no default value.
	TmuxSession string

	// TmuxManager is an optional tmux manager for spawning agents.
	// If nil, a new manager will be created when needed.
	// This is primarily used for testing.
	TmuxManager *tmux.Manager
}

// Server is the MCP server for ticket management.
type Server struct {
	mcpServer     *mcp.Server
	store         *ticket.Store
	session       *Session
	config        *Config
	projectConfig *config.Config
	lifecycle     *lifecycle.Executor
	tmuxManager   *tmux.Manager
}

// NewServer creates a new MCP server with the given configuration.
func NewServer(cfg *Config) (*Server, error) {
	if cfg == nil {
		cfg = &Config{}
	}

	// Set tickets directory - derive from ProjectPath or require explicit setting
	ticketsDir := cfg.TicketsDir
	if ticketsDir == "" {
		if cfg.ProjectPath != "" {
			ticketsDir = filepath.Join(cfg.ProjectPath, ".cortex", "tickets")
		} else {
			return nil, fmt.Errorf("MCP server requires CORTEX_PROJECT_PATH or CORTEX_TICKETS_DIR to be set")
		}
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

	// Load project config if ProjectPath is set
	var projectCfg *config.Config
	if cfg.ProjectPath != "" {
		var err error
		projectCfg, err = config.Load(cfg.ProjectPath)
		if err != nil {
			return nil, err
		}
	}

	// Create lifecycle executor
	lifecycleExec := lifecycle.NewExecutor()

	// Create MCP server
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "cortex-mcp",
		Version: version.Version,
	}, nil)

	s := &Server{
		mcpServer:     mcpServer,
		store:         store,
		session:       session,
		config:        cfg,
		projectConfig: projectCfg,
		lifecycle:     lifecycleExec,
		tmuxManager:   cfg.TmuxManager,
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
