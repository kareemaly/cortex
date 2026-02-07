---
id: 7fc3e9b6-78b7-4bb7-a5e8-8ee1c5e37bb0
author: claude
type: ticket_done
created: 2026-01-26T18:12:50.161881Z
---
## SSE Event System for Real-Time TUI Updates

### Summary
Implemented a complete SSE (Server-Sent Events) system that enables the kanban TUI to auto-refresh when tickets change, eliminating the need for manual polling.

### Architecture
1. **Event Bus** (`internal/events/bus.go`) — In-process pub/sub keyed by project path. Uses buffered channels (cap 64) with non-blocking emit so slow consumers never block the store.
2. **Store Integration** (`internal/ticket/store.go`) — NewStore now accepts optional bus+projectPath. Emits 9 event types: TicketCreated, TicketUpdated, TicketDeleted, TicketMoved, SessionStarted, SessionEnded, SessionStatus, CommentAdded, ReviewRequested.
3. **SSE Endpoint** (`internal/daemon/api/events.go`) — GET /events streams JSON events per project. Sets proper SSE headers, listens on request context for cleanup.
4. **SDK Client** (`internal/cli/sdk/client.go`) — SubscribeEvents() opens a long-lived SSE connection with a dedicated no-timeout HTTP client, parses data: lines, returns a channel.
5. **TUI Integration** (`internal/cli/tui/kanban/model.go`) — Subscribes on Init(), waits for events, reloads tickets on any event. Cancels SSE context on quit. Gracefully degrades if connection fails.
6. **Ticket Detail** (`internal/cli/tui/ticket/model.go`) — Added RefreshMsg and TicketID() accessor so the kanban can forward refresh events to the detail view.

### Key Decisions
- **Non-blocking emit**: Slow TUI consumers drop events rather than blocking the store. Since the TUI re-fetches all data on any event, missing intermediates is fine.
- **Nil bus for MCP**: MCP-created stores pass nil bus since they are separate from daemon-managed stores and do not need event emission.
- **Graceful degradation**: If SSE connection fails, the TUI silently falls back to manual refresh (press r).
- **No heartbeat**: Kept SSE simple without keepalive pings. Connection drops are detected by the reader goroutine exiting.

### Files Changed (15)
**New:**
- internal/events/bus.go
- internal/events/bus_test.go
- internal/daemon/api/events.go

**Modified:**
- internal/ticket/store.go (bus + emit)
- internal/ticket/store_test.go (updated NewStore signature)
- internal/daemon/api/store_manager.go (thread bus)
- internal/daemon/api/deps.go (Bus field)
- internal/daemon/api/server.go (SSE route)
- internal/daemon/api/integration_test.go (updated signatures)
- internal/daemon/mcp/server.go (nil bus)
- internal/daemon/mcp/tools_test.go (updated signatures)
- cmd/cortexd/commands/serve.go (wire bus)
- internal/cli/sdk/client.go (SubscribeEvents)
- internal/cli/tui/kanban/model.go (SSE subscription)
- internal/cli/tui/ticket/model.go (RefreshMsg)

### Verification
- make build: clean
- make test: all unit tests pass (including 6 new bus tests)
- make lint: 0 issues
- API integration tests: all pass
- Merge to main: completed (resolved one conflict in client.go with FocusTicket from concurrent work)

### Follow-up Notes
- MCP integration tests have pre-existing failures (EOF on initialize) unrelated to this change.
- Could add SSE heartbeat/keepalive pings in the future if connection stability becomes an issue.
- Could add event filtering (e.g., subscribe to specific event types) if needed.