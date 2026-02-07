---
id: ac557e2b-de92-4db9-9920-521c203d7e1c
author: claude
type: done
created: 2026-02-06T10:21:45.310869Z
---
## Root Cause

`ticketHeight()` in `internal/cli/tui/kanban/column.go` wrapped titles at the full `titleWidth`, but `renderAllTickets()` wraps at `titleWidth - badgeWidth` for tickets with type badges (e.g., `[debug]`). This width mismatch caused `cursorYOffset()` to underestimate the cursor's actual Y position in rendered content, producing two regressions after the viewport migration (b113b4e9):

1. **Highlight disappears on scroll** — accumulated offset error made the viewport scroll to the wrong region.
2. **Bottom chevron stays visible at end** — `totalTicketHeight()` returned a value smaller than the viewport's actual content height, so the "more below" check stayed true.

## Resolution

Updated `ticketHeight()` to subtract the badge width from `titleWidth` when a ticket has a type, matching the wrapping logic in `renderAllTickets()`. Both `cursorYOffset()` and `totalTicketHeight()` call `ticketHeight()`, so they're fixed automatically.

## Changes

- `internal/cli/tui/kanban/column.go`: Added badge width calculation in `ticketHeight()` (5 lines added, 1 removed)

## Verification

- `make build` — compiles cleanly
- `make test` — all unit tests pass
- Commit: 6dc7090