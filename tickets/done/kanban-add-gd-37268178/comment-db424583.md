---
id: db424583-b9a3-444f-bac0-7f62da3ee6f5
author: claude
type: ticket_done
created: 2026-01-27T13:50:12.223863Z
---
## Summary

Implemented two kanban TUI keybinding changes: removed the "a" approve shortcut (belongs in ticket detail view only) and added a "gd" (go daemon) two-key sequence to focus the CortexDaemon dashboard window via the HTTP API.

## Changes Made

### Part 1: Removed "a" Approve Shortcut
- Deleted `KeyApprove` constant from `keys.go`
- Removed `[a]pprove` from the help bar text
- Deleted `SessionApprovedMsg` and `ApproveErrorMsg` message types from `model.go`
- Removed the `SessionApprovedMsg` and `ApproveErrorMsg` cases from the `Update()` switch
- Removed the `if isKey(msg, KeyApprove)` handler block from `handleKeyMsg()`
- Deleted the `approveSession()` method entirely

### Part 2: Added "gd" Shortcut to Focus Daemon Dashboard
Full-stack implementation across 4 layers:

1. **Tmux layer** (`internal/tmux/window.go`): Added `FocusWindowByIndex` method mirroring the existing `KillWindowByIndex` pattern — checks session exists, runs `select-window -t {session}:{index}`
2. **API layer** (`internal/daemon/api/handlers.go` + `server.go`): Added `DaemonFocusHandler` as a global endpoint (no project header needed), registered as `POST /daemon/focus`
3. **SDK layer** (`internal/cli/sdk/client.go`): Added `FocusDaemonDashboard()` method using `httpClient.Do` directly (no project header), matching the `Health()` pattern
4. **TUI layer** (`internal/cli/tui/kanban/keys.go` + `model.go`): Added `KeyD` constant, inserted `gd` check between the `g` handler and the `pendingG = false` fallthrough, added `focusDaemonDashboard()` command reusing existing `FocusSuccessMsg`/`FocusErrorMsg` types

## Key Decisions
- **DaemonFocusHandler is a standalone function** (not a method on a handler struct) since it's a global endpoint that only needs the tmux manager, matching the pattern of `HealthHandler` and `ProjectsHandler`
- **Reused existing FocusSuccessMsg/FocusErrorMsg** instead of creating new message types, since the focus result semantics are identical
- **gd handler placed before pendingG=false fallthrough** to intercept the `d` key while `pendingG` is true, before the state is cleared

## Files Modified
| File | Change |
|------|--------|
| `internal/tmux/window.go` | Added `FocusWindowByIndex` method |
| `internal/daemon/api/handlers.go` | Added `DaemonFocusHandler`, added tmux import |
| `internal/daemon/api/server.go` | Registered `POST /daemon/focus` route |
| `internal/cli/sdk/client.go` | Added `FocusDaemonDashboard()` method |
| `internal/cli/tui/kanban/keys.go` | Removed `KeyApprove`, added `KeyD`, updated help text |
| `internal/cli/tui/kanban/model.go` | Removed all approve code, added `gd` handler + `focusDaemonDashboard()` |

## Verification
- `make build` — passed
- `make lint` — passed (0 issues)
- `make test` — all unit tests passed