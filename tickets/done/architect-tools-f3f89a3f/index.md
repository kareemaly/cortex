---
id: f3f89a3f-eddb-4c6d-aa7b-17eda9acc9e6
title: Architect Tools Cleanup
type: ""
created: 2026-01-22T11:17:15Z
updated: 2026-01-22T11:17:15Z
---
Simplify architect MCP tools and fix spawnSession behavior.

## Changes

### 1. spawnSession should NOT auto-move ticket
Currently `spawnSession` moves ticket from backlog to progress. Remove this - the spawned agent should call `moveTicketToProgress` themselves.

### 2. Merge listTickets and searchTickets
Consolidate into single `listTickets` tool with optional filters:
- `status` - filter by backlog/progress/review/done
- `query` - search title/body text

### 3. Remove getSessionStatus
`readTicket` already returns full ticket with sessions. No need for separate tool.

## Final Architect Tools (7)

| Tool | Input |
|------|-------|
| `listTickets` | `status?`, `query?` |
| `readTicket` | `id` |
| `createTicket` | `title`, `body?` |
| `updateTicket` | `id`, `title?`, `body?` |
| `deleteTicket` | `id` |
| `moveTicket` | `id`, `status` |
| `spawnSession` | `ticket_id`, `agent?` |

## Files Affected

- `~/projects/cortex1/internal/daemon/mcp/tools_architect.go`
- `~/projects/cortex1/internal/daemon/mcp/types.go`

## Implementation

### Commits Pushed
- `452d219` refactor: simplify architect MCP tools from 9 to 7

### Key Files Changed
| File | Changes |
|------|---------|
| `internal/daemon/mcp/types.go` | Added `Query` field to `ListTicketsInput`, deleted `SearchTicketsInput`, `GetSessionStatusInput`, `GetSessionStatusOutput` |
| `internal/daemon/mcp/tools_architect.go` | Updated `listTickets` with query filter, removed `searchTickets` and `getSessionStatus` tools, removed auto-move from `spawnSession` |
| `internal/daemon/mcp/tools_test.go` | Replaced search/getSessionStatus tests with new query filter tests and `TestHandleSpawnSessionNoAutoMove` |
| `internal/daemon/mcp/integration_test.go` | Replaced search tests with `TestIntegration_ListTickets_WithQuery` and `TestIntegration_ListTickets_WithStatusAndQuery` |
| `cmd/cortex/commands/architect.go` | Updated architect prompt to reflect new 7-tool set |
| `CLAUDE.md` | Updated MCP tools documentation |

### Important Decisions
- Empty query returns all tickets (unlike old `searchTickets` which errored on empty query)
- Removed date filter support from merged `listTickets` (was in `searchTickets` but rarely used)
- `spawnSession` no longer auto-moves ticket - spawned agent should call `pickupTicket` to move to progress

### Scope Changes
- None - implemented as specified in original ticket