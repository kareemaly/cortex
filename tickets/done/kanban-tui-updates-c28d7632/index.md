---
id: c28d7632-a7cc-4d45-bb20-e882df400cc1
title: Kanban TUI Updates
type: ""
created: 2026-01-23T07:27:18Z
updated: 2026-01-23T07:27:18Z
---
Update kanban TUI to handle orphaned sessions with resume/fresh prompt.

## Context

This is a fresh project with no users. No backward compatibility needed. Breaking changes are fine. Do not accumulate tech debt.

## Changes

### Remove `a` shortcut
Architect is opened via `cortex architect` command. Remove the `a` keybinding from kanban.

### Update `s` spawn behavior
When user presses `s` on a ticket:

1. Call SDK to spawn (mode=normal)
2. If response indicates orphaned state, show prompt:
   - `[r]esume` - call SDK with mode=resume
   - `[f]resh` - call SDK with mode=fresh
   - `[c]ancel` - cancel operation
3. If active, show message "Session already active"
4. If normal/ended, spawn succeeds

All spawning goes through SDK → API → Daemon. Kanban does NOT do direct tmux operations.

### Prompt UI
Simple modal or inline prompt for orphan options. Text-based is fine.

## Files to Change

- `internal/cli/tui/kanban/model.go` - handle spawn response, prompt logic
- `internal/cli/tui/kanban/keys.go` - remove `a` keybinding

## Verification

```bash
make lint
make test
make build
make test-integration
```

## Implementation

### Commits Pushed
- `1e74009` feat: add orphaned session handling modal to kanban TUI

### Key Files Changed
- `internal/cli/sdk/client.go` - Added `APIError` type to preserve error codes from API, updated `parseError()` to return structured errors
- `internal/cli/tui/kanban/keys.go` - Removed `KeyArchitect`, added `KeyFresh`, `KeyCancel`, `KeyEscape` for modal handling
- `internal/cli/tui/kanban/model.go` - Added modal state fields, `OrphanedSessionMsg`, orphan detection in `spawnSession()`, modal key handler, modal rendering

### Important Decisions
- Reused `KeyRefresh` ('r') as resume key in modal context to avoid adding another key constant
- Modal replaces both status bar and help bar when shown, keeping UI simple
- Truncate ticket title at 30 chars in modal prompt for display

### Scope Changes
- Added `APIError` type to SDK client (not originally scoped but necessary for detecting orphaned sessions by error code)