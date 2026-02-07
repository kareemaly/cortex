---
id: 78bebc75-70d6-4403-a200-cc2647e0fa0e
author: claude
type: done
created: 2026-02-04T13:22:12.217265Z
---
## Completed: `cortex upgrade` Command

### Summary
Implemented a self-update command that upgrades cortex and cortexd binaries from GitHub releases.

### Files Created (830 lines)
- `internal/upgrade/github.go` - GitHub API client for fetching release info
- `internal/upgrade/checksum.go` - Download and SHA256 verification
- `internal/upgrade/binary.go` - Binary detection, backup, replacement, code signing
- `internal/upgrade/upgrade.go` - Main orchestration logic
- `cmd/cortex/commands/upgrade.go` - CLI command

### Features
- `cortex upgrade` - Upgrade to latest version
- `cortex upgrade --check` - Check if update available
- `cortex upgrade -v VERSION` - Upgrade to specific version
- Downloads and verifies SHA256 checksums
- Backs up existing binaries to `~/.cortex/backups/`
- Handles sudo for write-protected directories
- Code signs on macOS
- Stops/restarts daemon during upgrade
- "dev" version always allows upgrade

### Verification
- Build passes
- Lint passes (0 issues)
- All tests pass
- Merged to main and pushed to origin