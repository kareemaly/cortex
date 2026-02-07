---
id: 7f7172bf-18d5-4932-b8ce-f695cfd1c606
title: Add `cortex upgrade` command
type: work
created: 2026-02-04T12:47:43.095615Z
updated: 2026-02-04T13:22:12.218686Z
---
## Goal

Allow users to self-update cortex and cortexd binaries from GitHub releases with a single command.

## Requirements

### Core functionality

- Fetch latest release from GitHub API (`https://api.github.com/repos/kareemaly/cortex/releases/latest`)
- Compare current version with latest
- Download correct binary for OS/arch (darwin/linux, amd64/arm64)
- Verify SHA256 checksum using `checksums.txt` from release
- Backup current binaries before replacing (`~/.cortex/bin/*.backup`)
- Replace both `cortex` and `cortexd` binaries
- Handle sudo if installed in system location (`/usr/local/bin/`)

### Daemon restart after upgrade

Use the existing `internal/daemon/autostart` infrastructure:
1. Check if daemon is running via `autostart.IsDaemonRunning()`
2. If running: call `autostart.StopDaemon()` before replacing binaries
3. After successful upgrade: call `autostart.EnsureDaemonRunning()` to restart with new version

### Commands

- `cortex upgrade` — perform the upgrade
- `cortex upgrade --check` — just check if update available, don't install

### UX

- Show progress: downloading, verifying, backing up, upgrading, restarting daemon
- Prompt for sudo if needed (detect install location permissions)
- Clear success/failure messaging
- If already up to date, say so and exit

### Binary naming convention

Binaries in releases follow this pattern (matches install.sh):
- `cortex-darwin-amd64`, `cortex-darwin-arm64`
- `cortex-linux-amd64`, `cortex-linux-arm64`
- `cortexd-darwin-amd64`, `cortexd-darwin-arm64`
- `cortexd-linux-amd64`, `cortexd-linux-arm64`
- `checksums.txt`

## Out of scope

- Config migrations (handle via backward compatibility for now)
- Windows support
- Rollback command (backup is passive safety net)