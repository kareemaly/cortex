---
id: aba05669-46eb-4b1a-9091-4730b63f23f3
author: claude
type: ticket_done
created: 2026-01-27T10:37:43.704952Z
---
## Summary

Routed architect MCP mutation operations (create, update, delete, move ticket) through the daemon HTTP API instead of using the local ticket store directly. This ensures all ticket status changes emit SSE events that the kanban TUI receives for real-time updates.

## Root Cause

The architect MCP server (`cortexd mcp`) runs as a separate process from the daemon (`cortexd serve`). It created a local ticket store with a `nil` event bus at `internal/daemon/mcp/server.go:108`. Events emitted on this local in-memory bus never reached the daemon's bus where SSE subscribers (kanban TUI) are connected. As a result, architect MCP mutations produced no SSE events.

## Key Decisions

1. **Route mutations through daemon HTTP API via SDK client** — This mirrors the pattern already used by ticket sessions, which correctly emit SSE events. The SDK client discovers the ticket's current status via `GetTicketByID` then calls the appropriate status-specific endpoint.

2. **Keep reads on local store** — Read operations (`handleListTickets`, `handleReadTicket`) stay on the local file-based store since they don't need to emit events and file reads are always fresh.

3. **Fix StatusReview validation** — The `handleMoveTicket` status validation was missing `ticket.StatusReview`, preventing moves to the review column via the architect MCP.

## Files Modified

1. **`internal/cli/sdk/client.go`** (+108 lines) — Added three new SDK methods:
   - `UpdateTicket(id, title, body *string)` — GET by-id to discover status, then PUT
   - `DeleteTicket(id)` — GET by-id to discover status, then DELETE (expects 204)
   - `MoveTicket(id, toStatus)` — GET by-id to discover status, then POST move

2. **`internal/daemon/mcp/server.go`** (+3 lines) — Initialize SDK client for architect sessions after DaemonURL default is set (line 123)

3. **`internal/daemon/mcp/tools_architect.go`** (+13/-25 lines) — Updated 4 mutation handlers:
   - `handleCreateTicket` → `s.sdkClient.CreateTicket()` with `wrapSDKError`/`ticketResponseToOutput`
   - `handleUpdateTicket` → `s.sdkClient.UpdateTicket()` (also removed redundant `s.store.Get` call)
   - `handleDeleteTicket` → `s.sdkClient.DeleteTicket()`
   - `handleMoveTicket` → `s.sdkClient.MoveTicket()` + added `ticket.StatusReview` to validation

4. **`internal/daemon/mcp/tools_test.go`** (+5/-5 lines) — Updated 5 tests to use `setupArchitectWithDaemon` since mutation handlers now require a daemon backend

## Verification

- `make build` — passes
- `make lint` — 0 issues
- `make test` — all tests pass (including all 42 MCP tests)