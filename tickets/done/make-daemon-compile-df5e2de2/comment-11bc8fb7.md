---
id: 11bc8fb7-291c-4ac3-8298-fc35918f66d5
author: claude
type: done
created: 2026-02-07T09:18:55.559263Z
---
## Summary

Made the daemon compile with the new frontmatter + directory-per-entity storage layer. This was a large refactor touching 31 files across the entire codebase.

### Key Changes

**Architecture:**
- Added `SessionManager` (`internal/daemon/api/session_manager.go`) — per-project session store manager mirroring StoreManager/DocsStoreManager pattern
- Added `TicketsConfig` to project config with configurable tickets path
- Fixed config defaults: `TicketsPath()` and `DocsPath()` now default to `{projectRoot}/tickets` and `{projectRoot}/docs` (project root, not `.cortex/`)

**Type Changes:**
- Flattened `TicketResponse` — removed nested `Dates` and `Session` structs; dates are top-level `Created`/`Updated`/`Due`
- Updated `SessionResponse` to match new `session.Session` (no `ID`, `EndedAt`, `CurrentStatus`)
- Removed `DatesResponse`, `StatusEntryResponse`
- `CommentResponse`: `Author` replaces `SessionID`, `Created` replaces `CreatedAt`
- Added `Tags` to `TicketResponse` and `TicketSummary`

**Spawn Package:**
- Split store interfaces: `StoreInterface` (just Get) + `SessionStoreInterface` (Create/End/GetByTicketID)
- Removed `StateEnded` — sessions are ephemeral (deleted on end = normal state)
- `DetectTicketState` accepts `*session.Session` directly

**Files Changed:**
- `cmd/cortex/commands/ticket_show.go`, `ticket_spawn.go` — flat dates, removed session display
- `cmd/cortexd/commands/serve.go` — wired SessionManager into deps + dispatcher
- `internal/cli/sdk/client.go` — removed stale type re-exports
- `internal/cli/tui/ticket/model.go`, `kanban/model.go`, `kanban/column.go`, `ticket/styles.go` — updated for flat fields
- `internal/core/spawn/` — orchestrate.go, spawn.go, state.go, spawn_test.go
- `internal/daemon/api/` — agent.go, deps.go, sessions.go, tickets.go, types.go, session_manager.go (new)
- `internal/daemon/mcp/` — types.go, tools_ticket.go, tools_architect.go, tools_test.go
- `internal/notifications/` — dispatcher.go, dispatcher_test.go
- `internal/project/config/config.go` — TicketsConfig, fixed defaults
- `internal/types/` — response.go, convert.go, convert_test.go
- `internal/ticket/ticket.go`, `internal/session/store_test.go`, `internal/storage/comment_test.go`

### Verification
- `make build` ✓
- `make lint` ✓ (0 issues)
- `make test` ✓ (all tests pass)
- Zero stale references to `ticket.Session`, `ticket.AgentStatus`, `ticket.StatusEntry`, `.Dates.`