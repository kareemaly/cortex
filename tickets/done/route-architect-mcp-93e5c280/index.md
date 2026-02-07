---
id: 93e5c280-bee0-49d5-9cba-d3098cc4f8a4
title: Route Architect MCP Reads Through Daemon HTTP API
type: ""
created: 2026-01-27T11:27:28.249661Z
updated: 2026-01-27T12:17:31.901564Z
---
## Problem

The architect MCP session has a hybrid data access pattern: mutations (create, update, delete, move) are routed through the daemon HTTP API via the SDK client, but reads (`listTickets`, `readTicket`) access the local ticket store directly. The `spawnSession` tool also reads the local store to look up ticket status before delegating to the HTTP API.

This breaks the "everything talks to the daemon" architecture. If the daemon runs on a remote VM, the architect MCP process wouldn't have access to the local ticket store files.

### Current Flow
```
Architect MCP reads  → local store (file system)
Architect MCP writes → SDK HTTP client → daemon API
```

### Expected Flow
```
Architect MCP reads  → SDK HTTP client → daemon API
Architect MCP writes → SDK HTTP client → daemon API
```

## Context

The broader architectural goal is that `cortexd` is the single source of truth and all clients (CLI, TUI, MCP sessions) communicate with it exclusively over HTTP. This enables:
- Running the daemon on a remote VM with agents and tmux
- Running local client TUIs (`cortex dashboard`, `cortex kanban`) pointed at the remote daemon
- Clean separation of concerns — no client needs direct file access

Currently all other clients (CLI, TUI, ticket MCP session) are already fully HTTP-routed. The architect MCP session is the only one with local store reads.

## Scope

### `internal/daemon/mcp/tools_architect.go`

- **`handleListTickets`** — Replace `s.store.List(status)` with SDK client calls (`s.sdkClient.ListTicketsByStatus(status, query)` or `s.sdkClient.ListAllTickets(query)`)
- **`handleReadTicket`** — Replace `s.store.Get(input.ID)` with `s.sdkClient.GetTicketByID(input.ID)` or `s.sdkClient.FindTicketByID(input.ID)`
- **`handleSpawnSession`** — Replace `s.store.Get(input.TicketID)` (used to look up ticket status for URL path) with `s.sdkClient.GetTicketByID(input.TicketID)` and extract status from the response

### `internal/daemon/mcp/server.go`

- Remove the local `store` field from the MCP server struct if it's no longer needed after routing all reads through HTTP
- Ensure `sdkClient` is always available for architect sessions

### Response mapping

- Map SDK client response types (`types.TicketResponse`, `types.TicketSummary`) to MCP output format
- Existing `ToTicketOutput` and `ToTicketSummary` helpers may need to be adapted to work with HTTP response types instead of local store types

## Acceptance Criteria

- [ ] `listTickets` routes through daemon HTTP API (no local store access)
- [ ] `readTicket` routes through daemon HTTP API (no local store access)
- [ ] `spawnSession` status lookup routes through daemon HTTP API
- [ ] Architect MCP server no longer requires a local `ticket.Store` reference
- [ ] All existing MCP tests pass (update test setup to use HTTP test server)
- [ ] `make build && make test` pass