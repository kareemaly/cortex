package upgrade

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// BinaryInfo contains information about the installed binaries.
type BinaryInfo struct {
	CortexPath  string
	CortexdPath string
	InstallDir  string
	NeedsSudo   bool
}

// DetectBinaryLocations finds the current cortex and cortexd binary locations.
func DetectBinaryLocations() (*BinaryInfo, error) {
	// Get current executable path
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	installDir := filepath.Dir(exe)
	cortexPath := exe
	cortexdPath := filepath.Join(installDir, "cortexd")

	// Verify cortexd exists
	if _, err := os.Stat(cortexdPath); err != nil {
		// Try to find cortexd in PATH
		cortexdPath, err = exec.LookPath("cortexd")
		if err != nil {
			return nil, fmt.Errorf("cortexd not found in same directory as cortex or in PATH")
		}
		// Update install dir to cortexd's location
		installDir = filepath.Dir(cortexdPath)
	}

	// Check if we need sudo to write to the install directory
	needsSudo := !isWritable(installDir)

	return &BinaryInfo{
		CortexPath:  cortexPath,
		CortexdPath: cortexdPath,
		InstallDir:  installDir,
		NeedsSudo:   needsSudo,
	}, nil
}

// BackupBinary creates a backup of a binary in the backup directory.
// Returns the backup path.
func BackupBinary(srcPath, backupDir string) (string, error) {
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	baseName := filepath.Base(srcPath)
	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(backupDir, fmt.Sprintf("%s.%s.bak", baseName, timestamp))

	// Read the source file
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to read binary: %w", err)
	}

	// Write to backup
	if err := os.WriteFile(backupPath, content, 0700); err != nil {
		return "", fmt.Errorf("failed to write backup: %w", err)
	}

	return backupPath, nil
}

// ReplaceBinary replaces a binary with a new one.
func ReplaceBinary(oldPath, newPath string, needsSudo bool) error {
	if needsSudo {
		cmd := exec.Command("sudo", "cp", newPath, oldPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to replace binary with sudo: %w", err)
		}

		// Set executable permissions
		cmd = exec.Command("sudo", "chmod", "+x", oldPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set permissions: %w", err)
		}
	} else {
		content, err := os.ReadFile(newPath)
		if err != nil {
			return fmt.Errorf("failed to read new binary: %w", err)
		}

		if err := os.WriteFile(oldPath, content, 0700); err != nil {
			return fmt.Errorf("failed to write binary: %w", err)
		}
	}

	return nil
}

// RestoreBackup restores a binary from a backup.
func RestoreBackup(originalPath, backupPath string, needsSudo bool) error {
	return ReplaceBinary(originalPath, backupPath, needsSudo)
}

// CodeSign signs a binary on macOS using ad-hoc signing.
func CodeSign(path string, needsSudo bool) error {
	if runtime.GOOS != "darwin" {
		return nil
	}

	var cmd *exec.Cmd
	if needsSudo {
		cmd = exec.Command("sudo", "codesign", "--force", "--sign", "-", path)
	} else {
		cmd = exec.Command("codesign", "--force", "--sign", "-", path)
	}

	// Ignore errors - codesign may fail on some systems but binaries still work
	_ = cmd.Run()

	return nil
}

// isWritable checks if a directory is writable by the current user.
func isWritable(dir string) bool {
	testFile := filepath.Join(dir, ".cortex-upgrade-test")
	f, err := os.Create(testFile)
	if err != nil {
		return false
	}
	_ = f.Close()
	_ = os.Remove(testFile)
	return true
}

// GetBackupDir returns the path to ~/.cortex/backups/.
func GetBackupDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".cortex", "backups"), nil
}

// GetBinaryName returns the platform-specific binary name.
func GetBinaryName(name string) string {
	return fmt.Sprintf("%s-%s-%s", name, runtime.GOOS, runtime.GOARCH)
}

// CleanupBackups removes old backup files, keeping only the most recent N backups.
func CleanupBackups(backupDir string, keepCount int) error {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Separate cortex and cortexd backups
	var cortexBackups, cortexdBackups []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if len(name) > 11 && name[:7] == "cortex." && name[len(name)-4:] == ".bak" {
			cortexBackups = append(cortexBackups, filepath.Join(backupDir, name))
		} else if len(name) > 12 && name[:8] == "cortexd." && name[len(name)-4:] == ".bak" {
			cortexdBackups = append(cortexdBackups, filepath.Join(backupDir, name))
		}
	}

	// Remove old backups (entries are already sorted by name, which includes timestamp)
	removeOldBackups(cortexBackups, keepCount)
	removeOldBackups(cortexdBackups, keepCount)

	return nil
}

func removeOldBackups(backups []string, keepCount int) {
	if len(backups) <= keepCount {
		return
	}
	// Remove oldest backups (first in the sorted list)
	for i := 0; i < len(backups)-keepCount; i++ {
		_ = os.Remove(backups[i])
	}
}
