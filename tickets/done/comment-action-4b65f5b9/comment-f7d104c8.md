---
id: f7d104c8-e988-48e4-bc90-dd6adb0222cc
author: claude
type: done
created: 2026-02-02T14:32:26.51764Z
---
## Summary

Implemented comment action execution via daemon with tmux popup support. This feature enables the TUI to execute actions attached to review comments (specifically `git_diff` actions) by opening a tmux popup in the ticket's session.

## Changes Made

### New Files
- `internal/tmux/popup.go` - Added `DisplayPopup()` method for opening tmux popups

### Modified Files
- `internal/install/install.go` - Detect lazygit on init, set `git_diff_tool` config
- `internal/daemon/api/server.go` - Added execute action route
- `internal/daemon/api/tickets.go` - Added `ExecuteAction` handler
- `internal/daemon/api/types.go` - Added `ExecuteActionResponse` type
- `internal/cli/sdk/client.go` - Added `ExecuteCommentAction()` method
- `internal/cli/tui/ticket/keys.go` - Added `KeyDiff`, updated help text
- `internal/cli/tui/ticket/model.go` - Handle `d` key, added diff messages

## How It Works
1. When viewing a review comment with a `git_diff` action in the TUI modal, press `d`
2. TUI calls SDK `ExecuteCommentAction()` → daemon API
3. Daemon loads `git_diff_tool` from global config (lazygit or git)
4. Opens tmux popup with the appropriate command in the ticket's repo path
5. Popup auto-closes when user exits the diff viewer

## Commit
`8cf9f1d` - feat(tui): add comment action execution with tmux popup