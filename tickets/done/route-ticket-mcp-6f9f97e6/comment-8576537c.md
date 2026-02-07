---
id: 8576537c-9d26-4f3a-855c-306f23414016
author: claude
type: ticket_done
created: 2026-01-26T17:27:44.013031Z
---
## Summary

Refactored spawned ticket agent MCP processes to call the daemon's HTTP API instead of creating their own `ticket.Store`. All ticket mutations now flow through the daemon, enabling future event bus integration.

**Before:** `cortexd mcp` (ticket session) → own `ticket.Store` → direct file writes
**After:** `cortexd mcp` (ticket session) → SDK HTTP client → daemon API → daemon's Store

Architect sessions remain unchanged (they run in-process with the daemon).

## Key Decisions

1. **Fail fast, no fallback:** Ticket sessions always require `CORTEX_DAEMON_URL`. If missing, `NewServer()` returns an error immediately. No legacy fallback to local store — we're in early development, breaking changes are fine.

2. **Conclude cleanup in daemon API:** The `POST /tickets/{id}/conclude` handler replicates the full conclude logic (add done comment → end session → move to done → cleanup worktree → kill tmux window). This centralizes cleanup in the daemon where it has access to all dependencies.

3. **Tests use httptest.NewServer:** Ticket tool tests spin up a real HTTP test server with the daemon API router, creating an end-to-end test path: MCP handler → SDK client → HTTP API → ticket store.

## Files Modified

| File | Change |
|------|--------|
| `internal/daemon/api/types.go` | Added 6 request/response types (AddComment, RequestReview, ConcludeSession) |
| `internal/daemon/api/tickets.go` | Added 4 handlers: GetByID, AddComment, RequestReview, Conclude |
| `internal/daemon/api/server.go` | Registered 4 new routes under /tickets |
| `internal/cli/sdk/client.go` | Added 4 methods + 3 response types: GetTicketByID, AddComment, RequestReview, ConcludeSession |
| `internal/core/spawn/config.go` | Added CORTEX_DAEMON_URL to MCP env config |
| `cmd/cortexd/commands/mcp.go` | Read CORTEX_DAEMON_URL from env, pass to MCP Config |
| `internal/daemon/mcp/server.go` | Added DaemonURL to Config, sdkClient to Server, conditional init (SDK for ticket, store for architect) |
| `internal/daemon/mcp/tools_ticket.go` | Rewrote all 4 handlers to use SDK client, added conversion helpers and SDK error wrapping |
| `internal/daemon/mcp/server_test.go` | Updated setupTestServer for ticket sessions, added TestNewServerTicketRequiresDaemonURL |
| `internal/daemon/mcp/tools_test.go` | Rewrote setupTicketSession to use httptest server, updated ticket tool tests |
| `internal/core/spawn/spawn_test.go` | Added CORTEX_DAEMON_URL assertion to MCP config test |

## Verification

- `make build` — compiles cleanly
- `make test` — all tests pass (existing + new)
- `make lint` — 0 issues