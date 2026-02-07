---
id: 5e5bcd27-92d7-4507-980b-6018b26066d7
author: claude
type: done
created: 2026-02-05T08:09:04.254792Z
---
## Summary

Implemented due date support for tickets with full stack support including data model, API, SDK, MCP tools, TUI indicators, and prompt injection.

### Changes Made

**Data Model**
- Added `DueDate *time.Time` to ticket `Dates` struct in `internal/ticket/ticket.go`

**Store Layer**
- Updated `Create()` to accept optional `dueDate` parameter
- Added `SetDueDate(id, dueDate)` method
- Added `ClearDueDate(id)` method

**Response Types**
- Added `DueDate` to `DatesResponse` and `TicketSummary` structs
- Updated conversion functions to include DueDate

**API Endpoints**
- `POST /tickets` now accepts `due_date` in RFC3339 format
- `PATCH /tickets/{id}/due-date` - Set due date
- `DELETE /tickets/{id}/due-date` - Clear due date
- `GET /tickets` and `GET /tickets/{status}` now support `due_before` query filter

**SDK Client**
- Updated `CreateTicket()` to accept `dueDate *time.Time`
- Added `SetDueDate(ticketID, dueDate)` method
- Added `ClearDueDate(ticketID)` method
- Updated `ListAllTickets()` and `ListTicketsByStatus()` to accept `dueBefore` filter

**MCP Tools (Architect)**
- `createTicket` - now accepts `due_date` parameter (RFC3339)
- `listTickets` - now accepts `due_before` filter (RFC3339)
- `updateDueDate` - new tool to set/update due date
- `clearDueDate` - new tool to remove due date

**TUI**
- Kanban view shows `[OVERDUE]` (red) or `[DUE SOON]` (yellow) badges on tickets
- Ticket detail view shows color-coded due date in attributes panel

**Prompts**
- Added `CurrentDate` to `ArchitectKickoffVars`
- Architect kickoff prompt now displays current date

**Documentation**
- Updated CLAUDE.md MCP tools table with new tools

### Files Changed
- `internal/ticket/ticket.go`
- `internal/ticket/store.go`
- `internal/types/response.go`
- `internal/types/convert.go`
- `internal/daemon/api/types.go`
- `internal/daemon/api/tickets.go`
- `internal/daemon/api/server.go`
- `internal/cli/sdk/client.go`
- `internal/daemon/mcp/types.go`
- `internal/daemon/mcp/tools_architect.go`
- `internal/prompt/template.go`
- `internal/core/spawn/spawn.go`
- `internal/install/defaults/claude-code/prompts/architect/KICKOFF.md`
- `internal/cli/tui/kanban/styles.go`
- `internal/cli/tui/kanban/column.go`
- `internal/cli/tui/ticket/styles.go`
- `internal/cli/tui/ticket/model.go`
- `cmd/cortex/commands/ticket_list.go`
- `internal/cli/tui/kanban/model.go`
- `internal/cli/tui/dashboard/model.go`
- `CLAUDE.md`
- Test files updated for new function signatures

### Verification
- `make build` ✅
- `make lint` ✅ (0 issues)
- `make test` ✅ (all tests passed)
- Pushed to origin/main