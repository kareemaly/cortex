package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kareemaly/cortex/internal/daemon/api"
	"github.com/kareemaly/cortex/internal/daemon/config"
	"github.com/kareemaly/cortex/internal/daemon/logging"
	"github.com/kareemaly/cortex/internal/tmux"
	"github.com/kareemaly/cortex/pkg/version"
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

	// Create StoreManager (stores are created per-project on demand)
	storeManager := api.NewStoreManager(logger)

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

	// Build dependencies
	deps := &api.Dependencies{
		StoreManager: storeManager,
		TmuxManager:  tmuxManager,
		Logger:       logger,
	}

	// Create and run server
	server := api.NewServer(cfg.Port, logger, deps)
	return server.Run(ctx)
}
