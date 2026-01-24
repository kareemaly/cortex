# Window Exists Session Error Handling

## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

`WindowExists` in `internal/tmux/window.go` returns an error when the tmux session doesn't exist. It only handles `WindowNotFoundError`, not `SessionNotFoundError`.

This causes "failed to detect architect state" when spawning architect after manually deleting the tmux session.

## Requirements

- `WindowExists` should return `false, nil` when the session doesn't exist (not just when the window doesn't exist)

## Implementation

### Commits
- `12a45c6` fix: WindowExists returns false when session missing

### Key Files Changed
- `internal/tmux/window.go` - Added `IsSessionNotFound(err)` check to `WindowExists`

### Changes
Added `|| IsSessionNotFound(err)` to the error handling in `WindowExists` so that when the tmux session itself doesn't exist, the function returns `false, nil` instead of bubbling up the error. This matches the existing behavior for `WindowNotFoundError`.

### Scope
No scope changes - implementation matched the original ticket requirements exactly.
