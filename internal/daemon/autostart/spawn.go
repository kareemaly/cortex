package autostart

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kareemaly/cortex/pkg/version"
)

const (
	// daemonExecutable is the name of the daemon binary.
	daemonExecutable = "cortexd"
)

// ErrDaemonNotFound is returned when cortexd is not found in PATH.
var ErrDaemonNotFound = errors.New("cortexd not found in PATH")

// SpawnDaemon starts cortexd as a detached background process.
// It writes the PID file and waits for the daemon to become healthy.
func SpawnDaemon(port int) error {
	// Find the daemon executable
	execPath, err := getDaemonExecutable()
	if err != nil {
		return err
	}

	// Prepare the command
	cmd := exec.Command(execPath)

	// Set platform-specific process attributes for detachment
	cmd.SysProcAttr = getSysProcAttr()

	// Detach from terminal - redirect to /dev/null
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Start the daemon
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Write PID file
	info := &PIDInfo{
		PID:       cmd.Process.Pid,
		Port:      port,
		StartedAt: time.Now(),
		Version:   version.Version,
	}
	if err := WritePIDFile(info); err != nil {
		// Try to kill the process if we can't write the PID file
		_ = cmd.Process.Kill()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	return nil
}

// StopDaemon sends SIGTERM to the daemon process and waits for it to exit.
// If the process doesn't exit within 5 seconds, it sends SIGKILL.
func StopDaemon() error {
	info, err := ReadPIDFile()
	if err != nil {
		if errors.Is(err, ErrNoPIDFile) {
			return fmt.Errorf("daemon is not running (no PID file)")
		}
		return err
	}

	process, err := os.FindProcess(info.PID)
	if err != nil {
		_ = RemovePIDFile()
		return fmt.Errorf("daemon process not found (PID %d)", info.PID)
	}

	// Check if the process is actually running
	if !IsProcessRunning(info.PID) {
		_ = RemovePIDFile()
		return fmt.Errorf("daemon is not running (stale PID file)")
	}

	// Send SIGTERM
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// Process might have already exited
		_ = RemovePIDFile()
		return nil
	}

	// Wait for process to exit (up to 5 seconds)
	done := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	select {
	case <-done:
		// Process exited gracefully
	case <-time.After(5 * time.Second):
		// Force kill
		_ = process.Signal(syscall.SIGKILL)
		<-done
	}

	// Clean up PID file
	_ = RemovePIDFile()

	return nil
}

// getDaemonExecutable finds cortexd in PATH.
func getDaemonExecutable() (string, error) {
	// First check if there's a cortexd next to the current executable
	// (for local development)
	currentExe, err := os.Executable()
	if err == nil {
		localDaemon := filepath.Join(filepath.Dir(currentExe), daemonExecutable)
		if _, err := os.Stat(localDaemon); err == nil {
			return localDaemon, nil
		}
	}

	// Look in PATH
	path, err := exec.LookPath(daemonExecutable)
	if err != nil {
		return "", ErrDaemonNotFound
	}

	return path, nil
}

// LogFilePath returns the path to ~/.cortex/daemon.log.
func LogFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".cortex", "daemon.log"), nil
}
