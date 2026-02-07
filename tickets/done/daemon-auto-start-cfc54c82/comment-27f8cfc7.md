---
id: 27f8cfc7-ba1b-4b49-ad5f-218f63f69825
author: claude
type: done
created: 2026-02-04T13:05:12.96076Z
---
## Completed: Daemon Auto-Start and Management

### What was done

Replaced the manual `cortex start` command with automatic daemon management. The daemon is now auto-started transparently when any daemon-dependent command is run.

### New Package: `internal/daemon/autostart/`

- **pidfile.go**: PID file management with JSON format (PID, port, start time, version)
- **spawn_darwin.go / spawn_linux.go**: Platform-specific process detachment (Setpgid for macOS, Setsid for Linux)
- **spawn.go**: Daemon spawning/stopping with graceful shutdown (SIGTERM → SIGKILL)
- **autostart.go**: `EnsureDaemonRunning()` with health check and backoff retry (1s → 2s → 5s)

### New Commands

- `cortex daemon status` - show PID, port, version, uptime
- `cortex daemon stop` - stop the daemon
- `cortex daemon restart` - stop then restart
- `cortex daemon logs` - show last 50 lines
- `cortex daemon logs -f` - follow log output

### Updated Commands

Added `ensureDaemon()` auto-start to: architect, kanban, dashboard, projects, show, ticket list, ticket spawn, ticket show

### Updated `cortex init`

Now starts daemon at the end of initialization with status output.

### Removed

Deleted `cortex start` command (replaced by transparent auto-start).

### Files Changed

- 21 files changed, 753 insertions(+), 93 deletions(-)
- Created 6 new files in autostart package
- Created 5 new daemon command files
- Modified 8 existing command files
- Deleted start.go

### Verification

- Build: passes
- Lint: 0 issues
- Tests: all pass
- Merged to main and pushed to origin