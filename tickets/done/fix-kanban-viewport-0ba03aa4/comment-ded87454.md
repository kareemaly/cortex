---
id: ded87454-6b7d-4306-9f80-708c9ea3d691
author: claude
type: review_requested
created: 2026-02-06T10:20:57.038139Z
action:
    type: git_diff
    args:
        commit: 6dc7090
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Root Cause

`ticketHeight()` wrapped titles at the full `titleWidth`, but `renderAllTickets()` wraps at `titleWidth - badgeWidth` for tickets with type badges (e.g., `[debug]`). This mismatch caused `cursorYOffset()` to underestimate the cursor's Y position in rendered content, producing two bugs:

1. **Highlight disappears on scroll**: The viewport scrolled to the wrong region because the computed cursor offset was too low, accumulating error with each badged ticket.
2. **Bottom chevron stays visible at end**: `totalTicketHeight()` (which calls `ticketHeight()`) returned a value smaller than the viewport's actual `TotalLineCount()`, so the "more content below" check remained true.

## Fix

Updated `ticketHeight()` in `column.go` to subtract the badge width from `titleWidth` when the ticket has a type, matching the wrapping logic in `renderAllTickets()`. Both `cursorYOffset()` and `totalTicketHeight()` call `ticketHeight()`, so they're fixed automatically.

## Verification

- `make build` — compiles cleanly
- `make test` — all unit tests pass