---
id: 47b4c76c-d934-47d3-ae23-6a209a5866fb
author: claude
type: review_requested
created: 2026-02-02T14:30:54.545221Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Implemented comment action execution via daemon with tmux popup. This enables the TUI to execute actions attached to review comments (specifically `git_diff` actions) by opening a tmux popup in the ticket's session.

## Changes

### 1. Tmux Popup Support (NEW FILE)
- **File:** `internal/tmux/popup.go`
- Added `DisplayPopup(session, workingDir, command)` method to open a tmux popup that closes when the command exits

### 2. Lazygit Detection in `cortex init`
- **File:** `internal/install/install.go`
- Modified `setupGlobal()` to detect if `lazygit` is available via `exec.LookPath`
- Sets `git_diff_tool: lazygit` if available, otherwise `git_diff_tool: git`
- Only applies on fresh init (existing config is preserved)

### 3. API Endpoint
- **File:** `internal/daemon/api/server.go`
- Added route: `POST /{id}/comments/{comment_id}/execute`

- **File:** `internal/daemon/api/tickets.go`
- Added `ExecuteAction` handler that:
  - Gets ticket and finds comment by ID
  - Verifies comment has an action and ticket has active session
  - Loads global config for `git_diff_tool` setting
  - Validates repo_path exists
  - Builds command based on tool (lazygit or git diff)
  - Calls `TmuxManager.DisplayPopup()`

- **File:** `internal/daemon/api/types.go`
- Added `ExecuteActionResponse` type

### 4. SDK Client Method
- **File:** `internal/cli/sdk/client.go`
- Added `ExecuteCommentAction(ticketID, commentID)` method

### 5. TUI Integration
- **File:** `internal/cli/tui/ticket/keys.go`
- Added `KeyDiff = "d"`
- Updated `modalHelpText()` to show `[d]iff` for review comments with actions

- **File:** `internal/cli/tui/ticket/model.go`
- Added `DiffExecutedMsg` and `DiffErrorMsg` message types
- Added `executingDiff` state flag
- In `handleDetailModalKey()`, handle `d` key when `comment.Action.Type == "git_diff"`
- Added `executeDiffAction(commentID)` command that calls SDK

## Error Handling

- 404: Comment not found
- 400: No action on comment, unsupported action type, invalid repo_path
- 409: No active session
- 503: Tmux unavailable
- 500: Tmux command failure

## Verification

- `make test` - All tests pass
- `make build` - Compilation successful
- `make lint` - No issues