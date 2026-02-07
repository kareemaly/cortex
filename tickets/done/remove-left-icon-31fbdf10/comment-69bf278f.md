---
id: 69bf278f-16dc-4c0a-8cc6-dad95f983490
author: claude
type: ticket_done
created: 2026-01-27T10:01:46.33127Z
---
## Summary

Removed redundant left icon prefixes from kanban ticket card titles. The `> ` prefix on selected/focused cards and the agent status icon (`●`, `▶`, `○`, `⏸`, `✗`) prefix on unselected active-session cards were both removed. All title lines now use a uniform `"  "` (two-space) indent regardless of card state.

The agent status icon remains visible on the metadata line at the bottom of each card (e.g., `● Read · Jan 27`), so no information is lost. Selected tickets remain visually distinct via the `selectedTicketStyle` background/foreground color highlight.

## Key Decisions

- **Uniform indent over prefix icons:** The title area no longer changes layout based on selection or session state. This prevents text shifting when navigating between cards and keeps the visual hierarchy clean.
- **Minimal change:** Only the two `if` blocks controlling the prefix were removed. No other rendering logic was altered.

## Files Modified

- `internal/cli/tui/kanban/column.go` — Removed 8 lines (two `if` blocks and their bodies), added 2 lines (simplified loop variables from `lineIdx, line` to `_, line`). Net: -6 lines.

## Verification

- `make build` — compiles successfully
- `make lint` — 0 issues

## Follow-up

None required. The change is self-contained.