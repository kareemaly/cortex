package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kareemaly/cortex/internal/daemon/autostart"
	"github.com/kareemaly/cortex/pkg/version"
)

// Options configures the upgrade behavior.
type Options struct {
	CheckOnly bool   // Just check, don't upgrade
	Version   string // Specific version (empty = latest)
}

// Result contains the outcome of an upgrade operation.
type Result struct {
	CurrentVersion string
	LatestVersion  string
	WasUpgraded    bool
	AlreadyLatest  bool
}

// StepCallback is called for each step of the upgrade process.
type StepCallback func(step string, success bool, message string)

// Run performs the upgrade operation.
func Run(opts Options, callback StepCallback) (*Result, error) {
	result := &Result{
		CurrentVersion: version.Version,
	}

	// Detect binary locations
	binInfo, err := DetectBinaryLocations()
	if err != nil {
		return nil, fmt.Errorf("failed to detect binary locations: %w", err)
	}

	// Fetch release info
	var release *Release
	if opts.Version != "" {
		// Ensure version has v prefix
		v := opts.Version
		if !strings.HasPrefix(v, "v") {
			v = "v" + v
		}
		release, err = GetRelease(v)
	} else {
		release, err = GetLatestRelease()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release: %w", err)
	}

	result.LatestVersion = release.TagName

	// Compare versions
	if !opts.CheckOnly && !shouldUpgrade(result.CurrentVersion, result.LatestVersion) {
		result.AlreadyLatest = true
		return result, nil
	}

	// If check only, return early
	if opts.CheckOnly {
		return result, nil
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "cortex-upgrade-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	baseURL := GetDownloadBaseURL(release.TagName)

	// Download checksums
	callback("checksums", false, "Downloading checksums...")
	checksumsURL := fmt.Sprintf("%s/checksums.txt", baseURL)
	checksumsContent, err := DownloadString(checksumsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download checksums: %w", err)
	}
	checksums := ParseChecksums(checksumsContent)
	callback("checksums", true, "Downloaded checksums")

	// Download and verify cortex
	cortexBinary := GetBinaryName("cortex")
	cortexTmpPath := filepath.Join(tmpDir, "cortex")

	callback("download_cortex", false, fmt.Sprintf("Downloading %s...", cortexBinary))
	cortexURL := fmt.Sprintf("%s/%s", baseURL, cortexBinary)
	if err := Download(cortexURL, cortexTmpPath); err != nil {
		return nil, fmt.Errorf("failed to download cortex: %w", err)
	}
	callback("download_cortex", true, "Downloaded")

	callback("verify_cortex", false, "Verifying checksum...")
	if expected, ok := checksums[cortexBinary]; ok {
		if err := VerifyChecksum(cortexTmpPath, expected); err != nil {
			return nil, fmt.Errorf("cortex checksum verification failed: %w", err)
		}
		callback("verify_cortex", true, "Verified checksum")
	} else {
		callback("verify_cortex", true, "Checksum not found, skipping verification")
	}

	// Download and verify cortexd
	cortexdBinary := GetBinaryName("cortexd")
	cortexdTmpPath := filepath.Join(tmpDir, "cortexd")

	callback("download_cortexd", false, fmt.Sprintf("Downloading %s...", cortexdBinary))
	cortexdURL := fmt.Sprintf("%s/%s", baseURL, cortexdBinary)
	if err := Download(cortexdURL, cortexdTmpPath); err != nil {
		return nil, fmt.Errorf("failed to download cortexd: %w", err)
	}
	callback("download_cortexd", true, "Downloaded")

	callback("verify_cortexd", false, "Verifying checksum...")
	if expected, ok := checksums[cortexdBinary]; ok {
		if err := VerifyChecksum(cortexdTmpPath, expected); err != nil {
			return nil, fmt.Errorf("cortexd checksum verification failed: %w", err)
		}
		callback("verify_cortexd", true, "Verified checksum")
	} else {
		callback("verify_cortexd", true, "Checksum not found, skipping verification")
	}

	// Make downloaded files executable
	_ = os.Chmod(cortexTmpPath, 0755)
	_ = os.Chmod(cortexdTmpPath, 0755)

	// Stop daemon if running
	daemonWasRunning := autostart.IsDaemonRunning()
	if daemonWasRunning {
		callback("stop_daemon", false, "Stopping daemon...")
		if err := autostart.StopDaemon(); err != nil {
			// Not fatal - daemon might not be running
			callback("stop_daemon", true, "Daemon not running")
		} else {
			callback("stop_daemon", true, "Stopped daemon")
		}
	}

	// Get backup directory
	backupDir, err := GetBackupDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get backup directory: %w", err)
	}

	// Backup existing binaries
	callback("backup_cortex", false, "Backing up cortex...")
	cortexBackup, err := BackupBinary(binInfo.CortexPath, backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to backup cortex: %w", err)
	}
	callback("backup_cortex", true, "Backed up cortex")

	callback("backup_cortexd", false, "Backing up cortexd...")
	cortexdBackup, err := BackupBinary(binInfo.CortexdPath, backupDir)
	if err != nil {
		// Try to clean up cortex backup
		_ = os.Remove(cortexBackup)
		return nil, fmt.Errorf("failed to backup cortexd: %w", err)
	}
	callback("backup_cortexd", true, "Backed up cortexd")

	// Replace binaries
	callback("install_cortex", false, "Installing cortex...")
	if err := ReplaceBinary(binInfo.CortexPath, cortexTmpPath, binInfo.NeedsSudo); err != nil {
		// Restore from backup
		_ = RestoreBackup(binInfo.CortexPath, cortexBackup, binInfo.NeedsSudo)
		return nil, fmt.Errorf("failed to install cortex: %w", err)
	}
	callback("install_cortex", true, "Installed cortex")

	callback("install_cortexd", false, "Installing cortexd...")
	if err := ReplaceBinary(binInfo.CortexdPath, cortexdTmpPath, binInfo.NeedsSudo); err != nil {
		// Restore cortex from backup
		_ = RestoreBackup(binInfo.CortexPath, cortexBackup, binInfo.NeedsSudo)
		_ = RestoreBackup(binInfo.CortexdPath, cortexdBackup, binInfo.NeedsSudo)
		return nil, fmt.Errorf("failed to install cortexd: %w", err)
	}
	callback("install_cortexd", true, "Installed cortexd")

	// Code sign on macOS
	callback("codesign", false, "Code signing...")
	_ = CodeSign(binInfo.CortexPath, binInfo.NeedsSudo)
	_ = CodeSign(binInfo.CortexdPath, binInfo.NeedsSudo)
	callback("codesign", true, "Code signed (macOS)")

	// Start daemon if it was running before
	if daemonWasRunning {
		callback("start_daemon", false, "Starting daemon...")
		if err := autostart.EnsureDaemonRunning(); err != nil {
			callback("start_daemon", false, fmt.Sprintf("Failed to start daemon: %v", err))
		} else {
			callback("start_daemon", true, "Started daemon")
		}
	}

	// Clean up old backups (keep last 3)
	_ = CleanupBackups(backupDir, 3)

	result.WasUpgraded = true
	return result, nil
}

// shouldUpgrade determines if an upgrade should occur.
// Returns true if target version is newer than current, or if current is "dev".
func shouldUpgrade(current, target string) bool {
	// "dev" version always allows upgrade
	if current == "dev" {
		return true
	}

	// Compare versions
	return compareVersions(current, target) < 0
}

// compareVersions compares two semver versions.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareVersions(a, b string) int {
	// Strip v prefix
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")

	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	// Compare each part
	for i := 0; i < 3; i++ {
		aNum := 0
		bNum := 0

		if i < len(aParts) {
			aNum, _ = strconv.Atoi(strings.Split(aParts[i], "-")[0])
		}
		if i < len(bParts) {
			bNum, _ = strconv.Atoi(strings.Split(bParts[i], "-")[0])
		}

		if aNum < bNum {
			return -1
		}
		if aNum > bNum {
			return 1
		}
	}

	return 0
}
