---
id: 6f9f97e6-31ea-4424-8924-85612f0ce4b6
title: Route Ticket MCP Tools Through Daemon HTTP API
type: ""
created: 2026-01-26T17:04:12.056489Z
updated: 2026-01-26T17:27:44.01664Z
---
## Context

This project is in early development. Breaking changes are fine. Do not accumulate tech debt — write clean, direct code without backwards-compatibility shims or unnecessary abstractions.

## Problem

Spawned ticket agent sessions run as **separate `cortexd mcp` processes** that create their own `ticket.Store` instance and write directly to the file system. This means mutations like `addComment`, `requestReview`, and `concludeSession` bypass the daemon entirely.

This is a blocker for the SSE Event System ticket — if mutations don't go through the daemon, the event bus will never see them.

### Current Flow (broken)

```
Ticket Agent → claude CLI → cortexd mcp (NEW PROCESS)
    → creates own ticket.Store
    → writes directly to .cortex/tickets/ files
    → daemon has no visibility
```

### Desired Flow

```
Ticket Agent → claude CLI → cortexd mcp (NEW PROCESS)
    → uses SDK HTTP client
    → calls daemon HTTP API (port 4200)
    → daemon's Store handles mutation
    → daemon can emit events
```

## Scope

### 1. Add Missing HTTP API Endpoints

The daemon HTTP API needs endpoints for ticket-session mutations that currently only exist as MCP tools:

- `POST /tickets/{id}/comments` — add a comment
- `POST /tickets/{id}/reviews` — request review
- `POST /tickets/{id}/conclude` — conclude session (end session, move to done)
- `POST /tickets/{id}/pickup` — pick up ticket (move to in_progress)

### 2. Add SDK Client Methods

Add corresponding methods to `internal/cli/sdk/client.go`:

- `AddComment(ticketID, sessionID, commentType, content)`
- `RequestReview(ticketID, sessionID, summary, filesChanged)`
- `ConcludeSession(ticketID, sessionID, report)`
- `PickupTicket(ticketID, sessionID)`

### 3. Refactor MCP Ticket Tools to Use SDK Client

In `internal/daemon/mcp/server.go` and `tools_ticket.go`:

- Stop creating a `ticket.Store` in `NewServer()` for ticket sessions
- Inject an SDK client instead (pass daemon URL via env var)
- Rewrite `handleAddTicketComment`, `handleRequestReview`, `handleConcludeSession`, `handlePickupTicket` to call the SDK client

### 4. Pass Daemon URL to Spawned MCP Process

In `internal/core/spawn/config.go`:

- Add `CORTEX_DAEMON_URL` (e.g. `http://localhost:4200`) to the MCP env config
- MCP server reads this on startup and creates an SDK client

### 5. Keep Read Path Local (Optional Optimization)

`readTicket` in ticket sessions can remain a direct file read for performance — it's not a mutation and doesn't need event emission. This is optional; routing reads through the daemon is also fine for consistency.

## Key Files

| File | Change |
|------|--------|
| `internal/daemon/api/tickets.go` | Add comment/review/conclude/pickup endpoints |
| `internal/daemon/api/routes.go` | Register new routes |
| `internal/cli/sdk/client.go` | Add SDK methods for new endpoints |
| `internal/daemon/mcp/server.go` | Accept SDK client instead of Store for ticket sessions |
| `internal/daemon/mcp/tools_ticket.go` | Rewrite handlers to use SDK client |
| `internal/core/spawn/config.go` | Add CORTEX_DAEMON_URL to env |
| `cmd/cortexd/commands/mcp.go` | Read CORTEX_DAEMON_URL, create SDK client |

## Acceptance Criteria

- [ ] All ticket MCP tool mutations route through daemon HTTP API
- [ ] New HTTP endpoints exist for comment, review, conclude, pickup
- [ ] SDK client has methods for all new endpoints
- [ ] MCP ticket tools use SDK client, not direct Store
- [ ] Daemon URL is passed to spawned MCP process via env
- [ ] Existing tests pass, new endpoints have test coverage
- [ ] Architect MCP session is unaffected (it runs in-process with the daemon)