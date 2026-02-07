---
id: b55e0bca-7bfd-4031-b5c6-29697013bc28
author: claude
type: ticket_done
created: 2026-01-27T12:17:31.899041Z
---
## Summary

Routed all 3 architect MCP read operations through the daemon HTTP API, eliminating the local `ticket.Store` dependency from the architect MCP session. This enforces the "everything talks to the daemon" architecture principle, enabling the daemon to run on a remote VM while all clients (CLI, TUI, MCP sessions) communicate exclusively over HTTP.

### Before
```
Architect MCP reads  → local store (file system)
Architect MCP writes → SDK HTTP client → daemon API
```

### After
```
Architect MCP reads  → SDK HTTP client → daemon API
Architect MCP writes → SDK HTTP client → daemon API
```

## Key Decisions

1. **Used `sdkClient.ListTicketsByStatus()` for list operations** — The daemon API already handles query filtering server-side, so no client-side filtering was needed. This simplified the handler from 20+ lines to ~10 lines.

2. **Used `sdkClient.GetTicketByID()` for read and spawn** — This status-agnostic endpoint avoids needing to know the ticket's status upfront, which was the main reason `handleSpawnSession` needed the local store.

3. **Added `ticketSummaryResponseToMCP()` conversion** — Maps the shared `types.TicketSummary` (which has `Updated`, `AgentStatus`, `AgentTool`) to the simpler MCP `TicketSummary` (just `ID`, `Title`, `Status`, `Created`, `HasActiveSession`).

4. **Used string literals for status validation** — Replaced `ticket.Status` constants with plain string comparisons (`"backlog"`, `"progress"`, etc.) to completely remove the `ticket` package import from handler code.

5. **Test migration via `storeManager.GetStore()`** — Tests still need direct store access for data setup (creating tickets, sessions). Rather than going through the HTTP API for setup, we get the underlying store from the daemon's `StoreManager`, which shares the same data the HTTP handlers access.

## Files Modified (7 files, +145/-318 lines)

| File | Changes |
|------|---------|
| `internal/daemon/mcp/tools_architect.go` | Rewrote 3 handlers to use SDK client; string literal status validation |
| `internal/daemon/mcp/types.go` | Added `ticketSummaryResponseToMCP`; removed `ToTicketOutput`, `ToTicketSummary`, `ToSessionOutput` |
| `internal/daemon/mcp/server.go` | Removed `store` field, `Store()` accessor, `TicketsDir` config; simplified constructor |
| `internal/daemon/mcp/errors.go` | Removed `WrapTicketError` function and `ticket` import |
| `cmd/cortexd/commands/mcp.go` | Removed `ticketsDir` variable and `TicketsDir` config |
| `internal/daemon/mcp/tools_test.go` | Migrated all architect tests to daemon-backed setup; removed `TestWrapTicketError` |
| `internal/daemon/mcp/server_test.go` | Removed `setupTestServer`, `TestServerStoreAccess`; updated constructor tests |

## Verification

- `make build` — compiles cleanly
- `make test` — all 46 MCP tests pass (plus all other package tests)
- `make lint` — 0 issues

## Follow-up Notes

- The `CORTEX_TICKETS_DIR` env var is now harmlessly ignored if still set by spawn config — no breaking change for existing deployments.
- The architect MCP session now requires `CORTEX_PROJECT_PATH` (previously could use `CORTEX_TICKETS_DIR` as alternative). This is the expected configuration path.