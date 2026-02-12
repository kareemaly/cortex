---
id: 9d64a7d6-e1b8-480b-b24a-9357818e5b54
author: claude
type: done
created: 2026-02-10T10:27:55.344699Z
---
## Summary

Implemented two TUI improvements across 7 files (+240/-52 lines):

### 1. Ticket TUI `e` keybinding
- Added `IndexPath(id)` to ticket store to resolve `index.md` path
- Added `POST /tickets/{id}/edit` API handler that opens `$EDITOR` (falls back to `vim`) in a tmux popup
- Added `EditTicket()` SDK client method
- Added `e` keybinding in ticket detail TUI (both body and comment list focus), with loading state and help text

### 2. Kanban tmux popup
- Added `POST /tickets/{id}/show-popup` API handler that runs `cortex show <id>` in a tmux popup
- Added `ShowTicketPopup()` SDK client method
- Replaced kanban's inline embedded `ticket.Model` with tmux popup — `o`/`Enter` now opens popup instead of embedding
- Removed unused `ticket` package import, `showDetail`/`detailModel` fields, `CloseDetailMsg` handling, detail delegation, and SSE forwarding

### Verification
- `make build` — passes
- `make lint` — 0 issues
- `make test` — all tests pass
- Pre-push hooks — all checks passed
- Merged to main and pushed