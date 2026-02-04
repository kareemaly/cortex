package autostart

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/kareemaly/cortex/internal/daemon/config"
)

// retryIntervals defines the backoff intervals for health check retries.
var retryIntervals = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	5 * time.Second,
}

// ErrDaemonStartFailed is returned when the daemon fails to start after retries.
var ErrDaemonStartFailed = errors.New("daemon failed to start after retries")

// EnsureDaemonRunning ensures the daemon is running, starting it if necessary.
// This is the main entry point for auto-starting the daemon.
func EnsureDaemonRunning() error {
	// Load config to get port
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	port := cfg.Port

	// Check if daemon is already healthy
	if checkHealth(port) {
		return nil
	}

	// Validate PID file - clean up stale PIDs
	_, err = ValidatePIDFile()
	if err != nil && !errors.Is(err, ErrNoPIDFile) && !errors.Is(err, ErrStalePID) {
		return fmt.Errorf("failed to validate PID file: %w", err)
	}

	// Spawn the daemon
	if err := SpawnDaemon(port); err != nil {
		return fmt.Errorf("failed to spawn daemon: %w", err)
	}

	// Wait for daemon to become healthy with retries
	if err := checkHealthWithRetry(port); err != nil {
		return err
	}

	return nil
}

// IsDaemonRunning returns true if the daemon is healthy.
func IsDaemonRunning() bool {
	cfg, err := config.Load()
	if err != nil {
		return false
	}
	return checkHealth(cfg.Port)
}

// checkHealth performs a single health check against the daemon.
func checkHealth(port int) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/health", port))
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode == http.StatusOK
}

// checkHealthWithRetry waits for the daemon to become healthy with backoff.
func checkHealthWithRetry(port int) error {
	for i, interval := range retryIntervals {
		// Wait before checking (except on first iteration for immediate check)
		if i > 0 {
			time.Sleep(interval)
		} else {
			// Small initial delay to let the daemon start
			time.Sleep(100 * time.Millisecond)
		}

		if checkHealth(port) {
			return nil
		}
	}

	// Final attempt after last interval
	time.Sleep(retryIntervals[len(retryIntervals)-1])
	if checkHealth(port) {
		return nil
	}

	return fmt.Errorf("%w: not healthy after %d attempts", ErrDaemonStartFailed, len(retryIntervals)+1)
}

// GetStatus returns the current daemon status including PID info and health.
func GetStatus() (*Status, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	status := &Status{
		Port: cfg.Port,
	}

	// Try to read PID file
	info, err := ReadPIDFile()
	if err != nil {
		if errors.Is(err, ErrNoPIDFile) {
			status.Running = false
			return status, nil
		}
		return nil, err
	}

	// Check if process is running
	if !IsProcessRunning(info.PID) {
		status.Running = false
		// Clean up stale PID file
		_ = RemovePIDFile()
		return status, nil
	}

	// Verify via health check
	status.Running = checkHealth(cfg.Port)
	if status.Running {
		status.PID = info.PID
		status.Version = info.Version
		status.Uptime = info.Uptime()
	} else {
		// Process exists but not healthy - might be starting up or crashed
		status.PID = info.PID
		status.Version = info.Version
	}

	return status, nil
}

// Status represents the current daemon status.
type Status struct {
	Running bool
	PID     int
	Port    int
	Version string
	Uptime  time.Duration
}
