---
id: 31eb4d6a-fa12-4f09-84ed-cd1bb163e332
author: claude
type: review_requested
created: 2026-02-18T07:22:07.671724Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
        commit: 189edc3
---
## Summary

Added `[x]` kill keybinding to the dashboard TUI for killing sessions directly from the project overview.

### Changes

**`internal/cli/tui/dashboard/keys.go`**
- Added `KeyKill Key = "x"` constant
- Updated help text to include `[x] kill`

**`internal/cli/tui/dashboard/model.go`**
- Added kill state fields to Model (`showKillConfirm`, `killProjectPath`, `killSessionID`, `killSessionName`, `killing`)
- Added `SessionKilledMsg` and `SessionKillErrorMsg` message types
- Added message handlers in `Update()` for both new messages
- Added kill confirmation modal handler in `handleKeyMsg()` (y/n/esc)
- Added `handleKillSession()` method with behavior matrix:
  - Session row + orphaned → kill immediately (no confirmation)
  - Session row + active → show confirmation dialog, then kill
  - Project row + orphaned architect → kill immediately
  - Project row + active architect → show confirmation dialog, then kill
  - Project row + no session → no-op
- Added `killSession()` command using existing `sdk.DefaultClient().KillSession()`
- Added kill confirmation dialog rendering in `View()`
- Added `m.killing` to the loading guard to block input during kill

### Verification
- `make build` — compiles cleanly
- `make lint` — 0 issues
- `make test` — all tests pass

### Reused existing infrastructure
- `sdk.DefaultClient(projectPath).KillSession(id)` — existing SDK method
- `warnBadgeStyle` — existing confirmation dialog style
- `clearStatusAfterDelay()` — existing 3s auto-clear
- SSE `SessionEnded` event auto-refreshes the UI after kill