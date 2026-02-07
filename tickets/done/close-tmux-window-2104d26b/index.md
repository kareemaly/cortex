---
id: 2104d26b-08fc-4b8b-953a-ede8eaccd574
title: Close Tmux Window on Session Conclude
type: ""
created: 2026-01-23T08:35:51Z
updated: 2026-01-23T08:35:51Z
---
## Problem

When a ticket agent calls `concludeSession` or `moveTicketToDone`, the tmux window remains open.

## Expected Behavior

The tmux window should close automatically when the session ends.

## Verification

```bash
make lint
make test
make build
make test-integration
```

## Implementation

### Commits Pushed

- `7d99c09` feat: close tmux window when ticket session concludes
- `8f2ec31` Merge branch 'ticket/2026-01-23-close-window-on-conclude'

### Key Files Changed

- `internal/daemon/mcp/server.go` - Added `killSessionWindow` helper method
- `internal/daemon/mcp/tools_ticket.go` - Modified `handleConcludeSession`, `handleMoveTicketToDone`, and `handleApprove` to call the helper

### Important Decisions

- Used best-effort cleanup pattern matching the existing HTTP API handler at `internal/daemon/api/sessions.go`
- Window kill errors are logged but not returned since cleanup is non-critical
- Session reference is saved before `EndSession()` call since the session data is needed for tmux window identification

### Scope Changes

None - implementation matches the original plan exactly.