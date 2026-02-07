---
id: 8c867218-0607-4a6e-a470-f2b4533b0b90
author: claude
type: ticket_done
created: 2026-01-27T09:45:51.602908Z
---
## Summary

Redesigned the kanban ticket card layout to move agent status (icon + tool name) from the title's first-line prefix to the bottom metadata line. This fixes two issues: agent status stealing ~12 characters of title width on line 1, and agent status disappearing entirely in the focused/selected state.

## Changes Made

### `internal/cli/tui/kanban/column.go` (1 file, +32/-18 lines)

1. **Replaced `formatAgentStatus` with two focused functions:**
   - `agentStatusIcon(t)` — returns the bare icon character (●, ▶, ○, ⏸, ✗) based on agent status
   - `agentStatusLabel(t)` — returns icon + truncated tool name as unstyled text for the metadata line

2. **Title prefix (unselected, active session):** Changed from `● TodoWrit ` (icon + tool) to just `● ` (icon only via `activeSessionStyle`), freeing the full column width for title text on all lines.

3. **Title prefix (selected):** Kept `> ` on line 1, `  ` on subsequent lines — no change.

4. **Metadata/date line:** Now renders agent status alongside the date when there's an active session:
   - Unselected: `● TodoWrit · Jan 27` (agent part in `activeSessionStyle`, rest in `ticketDateStyle`)
   - Selected: `● TodoWrit · Jan 27` (entire line in `selectedTicketStyle`)
   - No active session: `Jan 27` (unchanged behavior)

## Key Decisions

- **No changes to `styles.go`:** Existing styles (`activeSessionStyle`, `ticketDateStyle`, `selectedTicketStyle`) were sufficient for the new layout. No new styles needed.
- **Icon-only prefix on title line 1:** Kept the status icon (`●`) as a visual indicator on unselected tickets with active sessions, per the ticket scope. This preserves at-a-glance visibility of which tickets have agents without consuming title space for the tool name.
- **Unstyled label composition:** `agentStatusLabel` returns unstyled text so the caller controls styling — `activeSessionStyle` for unselected, inherited `selectedTicketStyle` for selected state.

## Verification

- `make build` — passes
- `make lint` — 0 issues
- `make test` — all unit tests pass

## Files Modified

- `internal/cli/tui/kanban/column.go`

## Follow-up Notes

None — all acceptance criteria met.