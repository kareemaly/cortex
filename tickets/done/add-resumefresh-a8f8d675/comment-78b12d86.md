---
id: 78b12d86-d091-4cf1-b40c-98f1b15acc72
author: claude
type: done
created: 2026-02-07T12:54:55.35978Z
---
## Summary

Added a resume/fresh mode selection modal for orphaned architect sessions in the dashboard TUI.

### Problem
When the architect session was orphaned, pressing `[s]` called `spawnArchitect("")` with no mode, triggering a 409 error. Users had no way to choose resume vs. fresh from the dashboard.

### Solution
Added an inline modal (following the existing unlink confirmation pattern) that appears when `[s]` or `[enter/f]` is pressed on a project with an orphaned architect. The modal offers `[r]esume`, `[f]resh`, or `[esc] cancel`.

### Changes
- **File**: `internal/cli/tui/dashboard/model.go` (66 insertions, 3 deletions)
  - Added `showArchitectModeModal` and `architectModeProjectPath` state fields
  - Added `handleArchitectModeKey()` modal key handler (r/f/esc)
  - Added `spawnArchitectWithMode()` command that passes explicit mode to `SpawnArchitect()`
  - Wired modal intercept in `handleKeyMsg` (same pattern as unlink confirmation)
  - Modified `handleSpawnArchitect` to detect orphaned state and show modal
  - Modified `handleFocusCurrentRow` orphaned branch to trigger modal instead of passive message
  - Added modal rendering in `View` with warn badge and help bar styles

### Verification
- `make build` — compiles cleanly
- `make lint` — 0 issues
- `make test` — all unit tests pass
- Pre-push hooks passed