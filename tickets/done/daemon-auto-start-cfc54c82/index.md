---
id: cfc54c82-5387-43f7-8ec6-d9712344f163
title: Daemon auto-start and management
type: work
created: 2026-02-04T12:45:58.680158Z
updated: 2026-02-04T13:05:12.963215Z
---
## Goal

Replace the current `cortex start` command with automatic daemon management. Any command that needs the daemon should auto-start it transparently.

## Requirements

### 1. Auto-start infrastructure

Create `EnsureDaemonRunning()` that:
- Checks daemon health via HTTP (port from `~/.cortex/settings.yaml`, default 4200)
- If not running: starts daemon in background with proper terminal detachment
- Uses PID file (`~/.cortex/daemon.pid`) for tracking
- Logs to `~/.cortex/daemon.log`
- Retries health check with backoff: 1s → 2s → 5s (3 attempts)
- Works reliably on both macOS and Linux

### 2. New `cortex daemon` subcommands

- `cortex daemon status` — show running state, PID, port, uptime
- `cortex daemon stop` — kill daemon process, remove PID file
- `cortex daemon restart` — stop then ensure running
- `cortex daemon logs` — show last ~50 lines of daemon.log
- `cortex daemon logs -f` — follow/tail the log

### 3. Integrate auto-start into existing commands

Wire `EnsureDaemonRunning()` into all daemon-dependent commands:
- `architect`
- `ticket list`, `ticket spawn`, `ticket show`
- `show`
- `projects`
- `dashboard`
- `kanban`

### 4. Update `cortex init`

Call `EnsureDaemonRunning()` at the END of init (after global/project setup and registration complete, so settings.yaml exists).

### 5. Remove `cortex start`

Delete the command entirely. Update any references.

## Exploration needed

- Review current `cortex start` implementation
- Review how daemon-dependent commands currently work
- Understand cross-platform daemon spawning (setsid, process detachment)
- Check existing SDK client initialization patterns