---
id: 0b0c169f-c9e0-4bfc-af38-a5289f49f7cd
author: claude
type: ticket_done
created: 2026-01-27T09:35:07.756012Z
---
## Summary

Implemented orphaned session spawn error handling with a resume/fresh dialog in the TUI, fixing two issues:

1. **API error code mismatch (bug fix)**: The HTTP API returned `"state_conflict"` for all `StateError`s, but the SDK's `IsOrphanedSession()` checked for `"session_orphaned"`. This meant the kanban's existing orphan modal never triggered.

2. **No spawn in ticket detail view (feature)**: The detail view only supported kill/approve actions, not spawn. Added full spawn capability and orphan modal to the ticket detail view.

## Key Decisions

- **Reused existing patterns**: The orphan modal in the ticket detail view follows the same pattern as the kanban board's existing orphan modal (same key bindings: r/f/c, same flow).
- **canSpawn() logic**: Only allows spawning when the ticket is in `backlog` or `progress` status and has no active session. This matches the spawn semantics defined in the orchestration layer.
- **Help bar conditionally shows [s]pawn**: The spawn key only appears in the help bar when `canSpawn()` is true, keeping the UI clean.

## Files Modified

| File | Lines Changed | Description |
|------|--------------|-------------|
| `internal/daemon/api/tickets.go` | +7/-1 | Return `"session_orphaned"` error code for orphaned StateErrors instead of generic `"state_conflict"` |
| `internal/cli/tui/ticket/keys.go` | +7/-1 | Added `KeySpawn`, `KeyFresh`, `KeyCancel` constants; updated `helpText()` with `canSpawn` parameter |
| `internal/cli/tui/ticket/model.go` | +145/-19 | Added spawn capability, orphan modal, new message types, key handling, and view rendering |

## Verification

- `make build` — compiles successfully
- `make lint` — 0 issues
- `make test` — all unit tests pass

## Follow-up Notes

- The MCP layer (`internal/daemon/mcp/`) has its own error handling for orphaned states (`ErrorCodeStateConflict`) which is separate from the HTTP API. No changes were needed there.
- Manual testing recommended: run `cortex kanban`, navigate to a ticket with an orphaned session, press `s` to verify the dialog appears. Also test from the detail view (press `o` then `s`).