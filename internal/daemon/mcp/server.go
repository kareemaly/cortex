package mcp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kareemaly/cortex/internal/cli/sdk"
	daemonconfig "github.com/kareemaly/cortex/internal/daemon/config"
	"github.com/kareemaly/cortex/internal/project/config"
	"github.com/kareemaly/cortex/internal/tmux"
	"github.com/kareemaly/cortex/pkg/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Config holds configuration for the MCP server.
type Config struct {
	// TicketID restricts the session to a specific ticket (ticket session).
	// If empty, the session is an architect session with full access.
	TicketID string

	// TicketType is the ticket type (work/debug/research/chore).
	// Only used for ticket sessions, to conditionally register tools.
	TicketType string

	// ProjectPath is the project root for hook execution.
	// If set, project config is loaded from this path.
	// Required for architect sessions.
	ProjectPath string

	// TmuxSession is the tmux session name for spawning agents.
	// Required for spawn operations - no default value.
	TmuxSession string

	// TmuxManager is an optional tmux manager for spawning agents.
	// If nil, a new manager will be created when needed.
	// This is primarily used for testing.
	TmuxManager *tmux.Manager

	// CortexdPath is an optional path to the cortexd binary.
	// If empty, binpath.FindCortexd() is used at runtime.
	// This is primarily used for testing.
	CortexdPath string

	// IsMeta indicates this is a meta session (global, above architects).
	// When true, TicketID and ProjectPath are not required.
	IsMeta bool

	// DaemonURL is the URL of the cortexd HTTP API.
	// When set for ticket sessions, the MCP server routes mutations through the daemon
	// instead of creating its own ticket store.
	DaemonURL string

	// Logger is an optional logger for warnings and errors.
	// If nil, warnings are silently ignored.
	Logger *slog.Logger
}

// Server is the MCP server for ticket management.
type Server struct {
	mcpServer     *mcp.Server
	sdkClient     *sdk.Client
	session       *Session
	config        *Config
	projectConfig *config.Config
	tmuxManager   *tmux.Manager
}

// NewServer creates a new MCP server with the given configuration.
func NewServer(cfg *Config) (*Server, error) {
	if cfg == nil {
		cfg = &Config{}
	}

	// Determine session type
	var session *Session
	if cfg.IsMeta {
		session = &Session{
			Type: SessionTypeMeta,
		}
	} else if cfg.TicketID != "" {
		session = &Session{
			Type:       SessionTypeTicket,
			TicketID:   cfg.TicketID,
			TicketType: cfg.TicketType,
		}
	} else {
		session = &Session{
			Type: SessionTypeArchitect,
		}
	}

	var sdkClient *sdk.Client
	var projectCfg *config.Config

	switch session.Type {
	case SessionTypeMeta:
		// Meta sessions are global — no project path required
		if cfg.DaemonURL == "" {
			cfg.DaemonURL = daemonconfig.DefaultDaemonURL
		}
		sdkClient = sdk.NewClient(cfg.DaemonURL, "")

	case SessionTypeTicket:
		// Ticket sessions always route through the daemon HTTP API
		if cfg.DaemonURL == "" {
			return nil, fmt.Errorf("ticket sessions require CORTEX_DAEMON_URL to be set")
		}
		sdkClient = sdk.NewClient(cfg.DaemonURL, cfg.ProjectPath)

	default:
		// Architect sessions route all operations through the daemon HTTP API
		if cfg.ProjectPath == "" {
			return nil, fmt.Errorf("MCP server requires CORTEX_PROJECT_PATH to be set")
		}

		var err error
		projectCfg, err = config.Load(cfg.ProjectPath)
		if err != nil {
			return nil, err
		}

		if cfg.DaemonURL == "" {
			cfg.DaemonURL = daemonconfig.DefaultDaemonURL
		}

		sdkClient = sdk.NewClient(cfg.DaemonURL, cfg.ProjectPath)
	}

	// Create MCP server
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "cortex-mcp",
		Version: version.Version,
	}, nil)

	s := &Server{
		mcpServer:     mcpServer,
		sdkClient:     sdkClient,
		session:       session,
		config:        cfg,
		projectConfig: projectCfg,
		tmuxManager:   cfg.TmuxManager,
	}

	// Register tools based on session type
	switch session.Type {
	case SessionTypeMeta:
		s.registerMetaTools()
	case SessionTypeArchitect:
		s.registerArchitectTools()
	default:
		s.registerTicketTools()
	}

	return s, nil
}

// Run starts the MCP server using stdio transport.
func (s *Server) Run(ctx context.Context) error {
	return s.mcpServer.Run(ctx, &mcp.StdioTransport{})
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
