package autostart

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// PIDInfo contains information about a running daemon.
type PIDInfo struct {
	PID       int       `json:"pid"`
	Port      int       `json:"port"`
	StartedAt time.Time `json:"started_at"`
	Version   string    `json:"version"`
}

// ErrNoPIDFile is returned when the PID file does not exist.
var ErrNoPIDFile = errors.New("no PID file found")

// ErrStalePID is returned when the PID file exists but the process is not running.
var ErrStalePID = errors.New("stale PID file")

// PIDFilePath returns the path to ~/.cortex/daemon.pid.
func PIDFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".cortex", "daemon.pid"), nil
}

// ReadPIDFile reads and parses the PID file.
// Returns ErrNoPIDFile if the file does not exist.
func ReadPIDFile() (*PIDInfo, error) {
	path, err := PIDFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoPIDFile
		}
		return nil, fmt.Errorf("failed to read PID file: %w", err)
	}

	var info PIDInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to parse PID file: %w", err)
	}

	return &info, nil
}

// WritePIDFile writes daemon information to the PID file.
func WritePIDFile(info *PIDInfo) error {
	path, err := PIDFilePath()
	if err != nil {
		return err
	}

	// Ensure the directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal PID info: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	return nil
}

// RemovePIDFile deletes the PID file.
func RemovePIDFile() error {
	path, err := PIDFilePath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}

	return nil
}

// IsProcessRunning checks if a process with the given PID is running.
func IsProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// ValidatePIDFile reads the PID file and validates that the process is still running.
// Returns the PIDInfo if valid, removes stale PID files.
func ValidatePIDFile() (*PIDInfo, error) {
	info, err := ReadPIDFile()
	if err != nil {
		return nil, err
	}

	if !IsProcessRunning(info.PID) {
		// Process is no longer running, clean up stale PID file
		_ = RemovePIDFile()
		return nil, ErrStalePID
	}

	return info, nil
}

// Uptime returns the duration since the daemon started.
func (p *PIDInfo) Uptime() time.Duration {
	return time.Since(p.StartedAt)
}
