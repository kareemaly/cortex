package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kareemaly/cortex1/internal/daemon/api"
	"github.com/kareemaly/cortex1/internal/daemon/config"
	"github.com/kareemaly/cortex1/internal/daemon/logging"
	"github.com/kareemaly/cortex1/pkg/version"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP API server",
	Long:  `Starts the Cortex HTTP API server for managing tickets and sessions.`,
	RunE:  runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Set serve as the default command when no subcommand is specified
	rootCmd.RunE = runServe
}

func runServe(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
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

	// Create and run server
	server := api.NewServer(cfg.Port, logger)
	return server.Run(ctx)
}
