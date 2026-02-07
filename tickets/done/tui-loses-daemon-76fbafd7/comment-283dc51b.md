---
id: 283dc51b-f164-45f7-afdd-60c55274b5e0
author: claude
type: review_requested
created: 2026-02-05T10:30:55.307494Z
action:
    type: git_diff
    args:
        commit: 07dda62
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Root Cause

The daemon's HTTP handler blocked indefinitely when `tmux display-popup -E` was executed because Go's `cmd.CombinedOutput()` waits for the process to exit. With lazygit running in a popup, this could block for minutes, causing the TUI's 10-second HTTP timeout to trigger with "context deadline exceeded".

**Call chain:**
1. TUI calls `ExecuteCommentAction` with 10s timeout
2. Daemon handler calls `DisplayPopup()` → `m.run()` → `cmd.CombinedOutput()`
3. `tmux display-popup -E` blocks until user closes lazygit
4. HTTP timeout → "context deadline exceeded"

## Fix Summary

Made `DisplayPopup` non-blocking by adding a `RunBackground` method that uses `cmd.Start()` instead of `cmd.CombinedOutput()`. The daemon now returns immediately after launching the popup.

### Changes:

1. **internal/tmux/tmux.go**:
   - Added `RunBackground(args ...string) error` to `TmuxRunner` interface
   - Implemented `RunBackground` in `execRunner` using `cmd.Start()`
   - Added `runBackground` method to `Manager`

2. **internal/tmux/popup.go**:
   - Changed `DisplayPopup` to use `runBackground()` instead of `run()`

3. **internal/tmux/mock_runner.go**:
   - Added `RunBackgroundFunc` field and `RunBackground` method to `MockRunner` for testing

## Verification

- `make test` - All tests pass
- `make lint` - No issues
- `make build` - Builds successfully