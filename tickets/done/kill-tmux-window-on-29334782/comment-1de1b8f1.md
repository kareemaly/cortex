---
id: 1de1b8f1-4664-4cb4-a4af-f452ae21402d
author: claude
type: ticket_done
created: 2026-01-26T15:58:50.322135Z
---
## Summary

Added tmux window cleanup to `handleConcludeSession` so the agent's tmux window is automatically killed after the session ends and the ticket moves to done.

## Changes Made

### Files Modified

1. **`internal/daemon/mcp/tools_ticket.go`**
   - Added `tmux` package import
   - Captured `tmuxWindow` from `t.Session.TmuxWindow` before `EndSession()` clears the session data
   - Added tmux window kill block after the existing worktree cleanup logic
   - Uses injected `s.tmuxManager` if available, otherwise falls back to `tmux.NewManager()`
   - Silently skips if tmux is not available, window name is empty, or `TmuxSession` is not configured
   - Suppresses `WindowNotFoundError` and `SessionNotFoundError` (expected race conditions where the window/session was already closed)
   - Logs warning only for unexpected errors

2. **`internal/tmux/mock_runner.go`**
   - Added `"kill-window"` to the switch cases in both `SetWindowExists` and `NewMockRunner` functions, alongside existing tmux commands that return empty success output

3. **`internal/daemon/mcp/tools_test.go`**
   - Added `TestHandleConcludeSession_KillsTmuxWindow` test that:
     - Creates a ticket with an active session and tmux window name
     - Customizes the mock runner to return the correct window name in list-windows output
     - Calls `handleConcludeSession` with a full report
     - Verifies the mock runner received a `kill-window` call
     - Verifies the ticket moved to done status

## Key Decisions

- **Pattern consistency**: Used the same tmux manager resolution pattern as `tools_architect.go` (use injected manager if available, otherwise create new one)
- **Error suppression**: Silently ignore `WindowNotFound` and `SessionNotFound` errors since the window may have already been closed by the user or another process — matches the pattern in `sessions.go:51-55`
- **Capture before EndSession**: Session info (including `TmuxWindow`) is captured before `EndSession()` is called, since that method clears the session data
- **Non-blocking**: Window kill failures don't fail the overall conclude operation — the main work (ending session, moving ticket to done) is already complete

## Verification

- `make test` — All unit tests pass including the new test
- `make lint` — 0 issues