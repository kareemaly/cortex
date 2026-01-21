package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/kareemaly/cortex1/internal/daemon/api"
	"github.com/kareemaly/cortex1/internal/daemon/config"
	"github.com/kareemaly/cortex1/internal/daemon/logging"
	"github.com/kareemaly/cortex1/internal/lifecycle"
	projectconfig "github.com/kareemaly/cortex1/internal/project/config"
	"github.com/kareemaly/cortex1/internal/ticket"
	"github.com/kareemaly/cortex1/internal/tmux"
	"github.com/kareemaly/cortex1/pkg/version"
	"github.com/spf13/cobra"
)

var servePort int

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP API server",
	Long:  `Starts the Cortex HTTP API server for managing tickets and sessions.`,
	RunE:  runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Register flags
	serveCmd.Flags().IntVar(&servePort, "port", 4200, "Port for the HTTP server")

	// Set serve as the default command when no subcommand is specified
	rootCmd.RunE = runServe
}

func runServe(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override port if flag was set
	if cmd.Flags().Changed("port") {
		cfg.Port = servePort
	}

	// Setup logging
	logger, err := logging.Setup(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("failed to setup logging: %w", err)
	}

	logger.Info("starting cortexd", "version", version.Version)

	// Create context that cancels on SIGINT or SIGTERM
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Info("received signal", "signal", sig.String())
		cancel()
	}()

	// Initialize ticket store
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	ticketsDir := filepath.Join(homeDir, ".cortex", "tickets")
	ticketStore, err := ticket.NewStore(ticketsDir)
	if err != nil {
		return fmt.Errorf("failed to create ticket store: %w", err)
	}

	// Load project config
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	var projectCfg *projectconfig.Config
	var projectRoot string

	projectCfg, projectRoot, err = projectconfig.LoadFromPath(cwd)
	if err != nil {
		if projectconfig.IsProjectNotFound(err) {
			logger.Warn("no .cortex directory found, using default config")
			projectCfg = projectconfig.DefaultConfig()
			projectRoot = cwd
		} else {
			return fmt.Errorf("failed to load project config: %w", err)
		}
	}

	logger.Info("loaded project config", "root", projectRoot, "agent", projectCfg.Agent)

	// Initialize tmux manager (nil if not installed)
	var tmuxManager *tmux.Manager
	tmuxManager, err = tmux.NewManager()
	if err != nil {
		if tmux.IsNotInstalled(err) {
			logger.Warn("tmux not installed, spawn functionality will be unavailable")
		} else {
			return fmt.Errorf("failed to initialize tmux: %w", err)
		}
	}

	// Initialize lifecycle executor
	hookExecutor := lifecycle.NewExecutor()

	// Build dependencies
	deps := &api.Dependencies{
		TicketStore:   ticketStore,
		ProjectConfig: projectCfg,
		ProjectRoot:   projectRoot,
		TmuxManager:   tmuxManager,
		HookExecutor:  hookExecutor,
		Logger:        logger,
	}

	// Create and run server
	server := api.NewServer(cfg.Port, logger, deps)
	return server.Run(ctx)
}
