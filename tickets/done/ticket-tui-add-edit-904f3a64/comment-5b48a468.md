---
id: 5b48a468-e5f9-45d1-ae3f-c06667e426ba
author: claude
type: review_requested
created: 2026-02-10T10:26:42.349266Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/2bb27de8-fc7c-44ad-81e1-561800144f75
---
## Summary

Implemented two related TUI improvements:

### 1. Ticket TUI `e` keybinding — edit ticket in `$EDITOR` via tmux popup
- Added `IndexPath(id)` method to ticket store (`internal/ticket/store.go`) to resolve a ticket's `index.md` filesystem path
- Added `Edit` API handler (`POST /tickets/{id}/edit`) that opens `$EDITOR` (falls back to `vim`) in a tmux popup
- Added `EditTicket()` SDK client method
- Added `KeyEdit = "e"` keybinding in both body and comment list focus modes
- Added `EditExecutedMsg`/`EditErrorMsg` message types with loading state
- Updated help text to show `[e]dit`

### 2. Kanban: replaced inline embedded detail view with tmux popup
- Added `ShowPopup` API handler (`POST /tickets/{id}/show-popup`) that runs `cortex show <id>` in a tmux popup
- Added `ShowTicketPopup()` SDK client method
- Removed embedded `ticket.Model` from kanban — no more `showDetail`/`detailModel` fields, `CloseDetailMsg` handling, detail model delegation, or SSE forwarding
- `o`/`Enter` now calls `ShowTicketPopup` instead of embedding the ticket detail view
- Added `PopupOpenedMsg`/`PopupErrorMsg` message types with status messages
- Removed unused `ticket` package import from kanban

### Files changed (7)
| File | Change |
|------|--------|
| `internal/ticket/store.go` | Added `IndexPath()` method |
| `internal/daemon/api/tickets.go` | Added `Edit` and `ShowPopup` handlers |
| `internal/daemon/api/server.go` | Registered 2 new routes |
| `internal/cli/sdk/client.go` | Added `EditTicket` and `ShowTicketPopup` methods |
| `internal/cli/tui/ticket/keys.go` | Added `KeyEdit`, updated help text |
| `internal/cli/tui/ticket/model.go` | Added `e` keybinding, edit messages, edit command |
| `internal/cli/tui/kanban/model.go` | Replaced inline detail with popup, removed embedded ticket model |

### Verification
- `make build` — compiles successfully
- `make lint` — 0 issues
- `make test` — all tests pass