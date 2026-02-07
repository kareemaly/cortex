---
id: b6f6d602-0d50-4c92-8815-ad7d5714300b
author: claude
type: review_requested
created: 2026-02-04T13:04:35.965078Z
action:
    type: git_diff
    args:
        commit: 7d8759c
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/b1f12b56-0c41-4dc6-8a6e-947614dae76c
---
## Summary

Implemented automatic daemon management to replace the manual `cortex start` command. The daemon is now auto-started transparently when any daemon-dependent command is run.

## Changes

### New Package: `internal/daemon/autostart/`

- **pidfile.go**: PID file management with JSON format storing PID, port, start time, and version
  - `PIDFilePath()`, `ReadPIDFile()`, `WritePIDFile()`, `RemovePIDFile()`
  - `IsProcessRunning()` uses signal 0 to check process existence
  - `ValidatePIDFile()` removes stale PID files automatically

- **spawn_darwin.go / spawn_linux.go**: Platform-specific process attributes
  - macOS: `Setpgid: true` creates new process group
  - Linux: `Setsid: true` creates new session

- **spawn.go**: Daemon spawning and stopping
  - `SpawnDaemon()` starts cortexd detached with proper terminal detachment
  - `StopDaemon()` sends SIGTERM, waits 5s, then SIGKILL if needed
  - `getDaemonExecutable()` finds cortexd (local or PATH)
  - `LogFilePath()` returns ~/.cortex/daemon.log

- **autostart.go**: Main entry point
  - `EnsureDaemonRunning()` - check health, validate PID, spawn if needed
  - `IsDaemonRunning()` - quick health check
  - `GetStatus()` - full status with uptime
  - Health check with backoff retry: 1s → 2s → 5s (3 attempts)

### New Commands: `cmd/cortex/commands/daemon*.go`

- **daemon.go**: Parent command grouping subcommands
- **daemon_status.go**: Shows PID, port, version, uptime
- **daemon_stop.go**: Stops daemon via StopDaemon()
- **daemon_restart.go**: Stop then EnsureDaemonRunning()
- **daemon_logs.go**: Shows last 50 lines, -f for follow mode

### Modified Commands

Added `ensureDaemon()` call at start of Run function:
- architect.go, kanban.go, dashboard.go, projects.go
- show.go, ticket_list.go, ticket_spawn.go, ticket_show.go

### Updated init.go

Added daemon startup at end of `runInit()`:
```
Daemon:
  ✓ Daemon running
```
or on failure:
```
Daemon:
  ✗ Failed to start daemon: ...
    Run 'cortex daemon restart' to try again
```

### Removed

- **start.go**: Deleted entirely, replaced by auto-start

## Testing

- ✅ `make build` - compiles successfully
- ✅ `make lint` - passes with 0 issues  
- ✅ `make test` - all unit tests pass

## Architecture

```
User runs command (e.g., cortex kanban)
    ↓
ensureDaemon() called → EnsureDaemonRunning()
    ↓
Check health: GET http://localhost:{port}/health
    ↓ If not running:
    ValidatePIDFile() - clean up stale PIDs
    SpawnDaemon() - start detached, write PID file
    checkHealthWithRetry() - wait with backoff
    ↓
Continue with original command
```