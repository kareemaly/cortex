---
id: 22ec4543-7a85-46e3-be6a-73c072f1d5ba
author: claude
type: done
created: 2026-02-05T10:32:41.027171Z
---
## Root Cause

The daemon's HTTP handler blocked indefinitely when `tmux display-popup -E` was executed because Go's `cmd.CombinedOutput()` waits for the child process to exit. When lazygit or other interactive programs ran in a popup, the handler would block until the user closed the popup, causing the TUI's 10-second HTTP timeout to trigger with "context deadline exceeded".

**Call chain:**
1. TUI calls `ExecuteCommentAction` with 10s HTTP timeout
2. Daemon handler calls `DisplayPopup()` → `m.run()` → `cmd.CombinedOutput()`
3. `tmux display-popup -E` blocks until user closes lazygit
4. HTTP timeout → "context deadline exceeded" → TUI shows error

## Resolution

Made `DisplayPopup` non-blocking by using `cmd.Start()` instead of `cmd.CombinedOutput()`. The daemon now returns immediately after launching the popup process.

## Files Changed

1. **internal/tmux/tmux.go**
   - Added `RunBackground(args ...string) error` to `TmuxRunner` interface
   - Implemented `RunBackground` in `execRunner` using `cmd.Start()`
   - Added `runBackground` method to `Manager`

2. **internal/tmux/popup.go**
   - Changed `DisplayPopup` to call `runBackground()` instead of `run()`

3. **internal/tmux/mock_runner.go**
   - Added `RunBackgroundFunc` field and `RunBackground` method to `MockRunner`

## Verification

- `make test` - All tests pass
- `make lint` - 0 issues
- `make build` - Builds successfully

## Commit

07dda62 fix: make DisplayPopup non-blocking to prevent HTTP timeout