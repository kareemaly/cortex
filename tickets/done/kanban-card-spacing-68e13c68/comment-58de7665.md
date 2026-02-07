---
id: 58de7665-82f5-4b9b-87e1-aa41b76791ad
author: claude
type: ticket_done
created: 2026-01-27T10:17:09.976857Z
---
## Summary

Implemented four kanban card spacing and layout fixes in `internal/cli/tui/kanban/column.go`:

### Changes Made

1. **Removed 2-char left indent from title and metadata lines**
   - Eliminated `prefix := "  "` from both selected and normal ticket rendering paths
   - Eliminated `meta := "  "` initialization from both paths
   - Adjusted `titleWidth` from `max(width-6, 10)` to `max(width-4, 10)` to account for the reclaimed space (still need 4 chars for lipgloss `Padding(0,1)` on ticket/column styles)

2. **Added vertical gap between ticket cards**
   - Changed inter-ticket separator from `"\n"` to `"\n\n"` to insert a blank line between cards for better visual separation

3. **Fixed top/bottom margin inconsistency (Done column first ticket)**
   - Top scroll indicator: Changed from unconditionally emitting a newline when `needsScrolling` to only emitting when `needsScrolling && c.scrollOffset > 0`
   - Bottom scroll indicator: Same pattern — only emits when `needsScrolling && c.scrollOffset+visibleCount < len(c.tickets)`
   - This eliminates the extra blank line above the first ticket when scrolling isn't active at the top

4. **Replaced fixed `linesPerTicket` with dynamic height calculation**
   - Deleted `const linesPerTicket = 6`
   - Added `ticketHeight()`: computes actual lines for a ticket (wrapped title lines + 1 metadata line)
   - Added `totalTicketHeight()`: sums all ticket heights including gap lines between them
   - Added `countVisibleTickets()`: counts how many tickets fit in available lines from a given start index
   - Moved `titleWidth` calculation before scroll logic so it's available for height computation
   - Added two-pass stabilization: compute visibleCount → EnsureCursorVisible → recompute if scrollOffset changed

### Key Decisions

- **Dynamic vs fixed height**: Replaced the hardcoded `linesPerTicket = 6` with actual height computation based on wrapped title text. This ensures tickets fill the full column height before scroll indicators appear, regardless of title length.
- **Two-pass stabilization**: After computing visibleCount, EnsureCursorVisible may change scrollOffset, which could change which tickets are visible and thus the height calculation. The second pass ensures consistency.
- **Gap lines accounted for in height**: The `countVisibleTickets` helper includes gap lines (1 per inter-ticket boundary) in its available space calculation, matching the actual render output.

### Files Modified

- `internal/cli/tui/kanban/column.go` — 61 insertions, 31 deletions

### Verification

- `make build` — compiles successfully
- `make lint` — 0 issues
- `make test` — all unit tests pass