---
id: eff820a8-9be5-42e3-9469-cc4be5c3893a
author: claude
type: ticket_done
created: 2026-01-27T13:42:12.88923Z
---
## Summary

Fixed dashboard focus not working when switching to agent sessions from the `cortex start` TUI.

### Root Cause

The dashboard runs in the `CortexDaemon` tmux session while project agents run in separate tmux sessions (named after the project). When focusing a session, the daemon called `tmux select-window -t project:N`, which selects the window within the project's session but does **not** switch the user's tmux client from `CortexDaemon` to `project`. The user sees nothing happen.

### Fix

Added `tmux switch-client -t <session>` after every `select-window` call. When run without `-c` (no specific client target), tmux targets the most recently active client — which is the user's terminal, since they just pressed a key in the dashboard.

### Key Decisions

- **Non-fatal SwitchClient calls**: All `SwitchClient` calls log warnings on failure rather than returning errors. This ensures `select-window` still succeeds even if no tmux client is attached (e.g., daemon running headless).
- **Non-interactive execution**: `SwitchClient` uses `m.run()` (not `RunInteractive`), making it safe for daemon-side execution without a TTY.
- **No interface changes needed**: `SwitchClient` is only called from API handlers which use `*tmux.Manager` directly (via `deps.TmuxManager`), so no interface updates were required in `spawn.TmuxManagerInterface`.

### Files Modified (6 files, +66/-2)

1. **`internal/tmux/session.go`** — Added `SwitchClient(session string) error` method
2. **`internal/daemon/api/tickets.go`** — Added `SwitchClient` after `FocusWindow` in Focus handler and Spawn already-active path
3. **`internal/daemon/api/architect.go`** — Added `SwitchClient` after `FocusWindow` in architect Spawn already-active path
4. **`internal/daemon/api/sessions.go`** — Added `SwitchClient` after `FocusWindow` in Approve handler
5. **`internal/tmux/mock_runner.go`** — Added `"switch-client"` to recognized no-op commands in both mock configurations
6. **`internal/tmux/tmux_test.go`** — Added `TestSwitchClient` with subtests for correct args and error propagation

### Verification

- `make lint` — 0 issues
- `make test` — all tests pass including new `TestSwitchClient`
- Merged to main via fast-forward