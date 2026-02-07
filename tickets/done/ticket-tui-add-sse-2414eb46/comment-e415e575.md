---
id: e415e575-1d5d-4a27-94fa-70ad11889bbd
author: claude
type: ticket_done
created: 2026-01-28T06:03:11.656699Z
---
## Summary

Added SSE (Server-Sent Events) subscription to the standalone ticket TUI (`cortex show <id>`) so it auto-refreshes when relevant events occur, matching the real-time behavior already present in the kanban board.

## Changes Made

### File Modified
- **`internal/cli/tui/ticket/model.go`** (+60 lines, -1 line)

### Specific Changes

1. **Added `"context"` import** — required for SSE context management.

2. **Added SSE fields to `Model` struct** — `eventCh <-chan sdk.Event` and `cancelEvents context.CancelFunc` to hold the SSE channel and cancellation function.

3. **Added SSE message types** — `sseConnectedMsg` (carries channel + cancel func) and `EventMsg` (signals a relevant event was received).

4. **Modified `Init()`** — When not embedded, batches `loadTicket()` + `subscribeEvents()`. When embedded (kanban host), only calls `loadTicket()` since kanban already handles SSE.

5. **Added SSE message handlers in `Update()`**:
   - `sseConnectedMsg`: stores channel/cancel, starts `waitForEvent()` loop
   - `EventMsg`: sets loading state, batches `loadTicket()` + next `waitForEvent()`

6. **Added SSE cancellation on quit** — calls `m.cancelEvents()` before `tea.Quit` in the standalone quit path to cleanly close the SSE connection.

7. **Added `subscribeEvents()` method** — creates a background context, calls `client.SubscribeEvents(ctx)`, returns `sseConnectedMsg`. On error, cancels context and returns nil (graceful degradation).

8. **Added `waitForEvent()` method** — reads from the event channel in a loop, filtering by ticket ID. Only returns `EventMsg{}` when `event.TicketID` matches the current ticket. Events for other tickets are silently consumed. Returns nil when channel closes.

## Key Decisions

- **Ticket ID filtering in `waitForEvent()`**: Unlike the kanban (which refreshes on all events), the ticket TUI filters events by ticket ID since it only displays a single ticket. This avoids unnecessary reloads.
- **Skip SSE when embedded**: The `embedded` flag (already present) is used to avoid duplicate SSE subscriptions when the ticket view is hosted inside the kanban board.
- **Graceful degradation**: If SSE connection fails, the TUI continues working with manual refresh (`r` key) only — no error displayed.
- **Same pattern as kanban**: Followed the exact same `subscribeEvents()` → `sseConnectedMsg` → `waitForEvent()` → `EventMsg` loop pattern from `internal/cli/tui/kanban/model.go` for consistency.

## Verification
- `make build` — compiled successfully
- `make lint` — 0 issues
- `make test` — all tests pass