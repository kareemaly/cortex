---
id: b0bec754-a262-4bcd-b207-90e4e1cca2a0
author: claude
type: review_requested
created: 2026-02-05T08:07:27.964618Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Implemented due date support for tickets with optional due date field, MCP tools, API endpoints, TUI indicators, and prompt injection.

### Changes Made

**Data Model (Phase 1)**
- `internal/ticket/ticket.go`: Added `DueDate *time.Time` to `Dates` struct

**Response Types (Phase 2)**
- `internal/types/response.go`: Added `DueDate` to `DatesResponse` and `TicketSummary`
- `internal/types/convert.go`: Updated `ToDatesResponse()` and `ToTicketSummary()` to include DueDate

**Store Layer (Phase 3)**
- `internal/ticket/store.go`: 
  - Updated `Create()` signature to accept `dueDate *time.Time`
  - Added `SetDueDate(id string, dueDate *time.Time)` method
  - Added `ClearDueDate(id string)` method

**API Layer (Phase 4)**
- `internal/daemon/api/types.go`: Added `DueDate` to `CreateTicketRequest`, added `SetDueDateRequest` struct, updated `filterSummaryList` for dueBefore filter
- `internal/daemon/api/tickets.go`: Added `SetDueDate` and `ClearDueDate` handlers, updated `Create`, `ListAll`, `ListByStatus` for due date support
- `internal/daemon/api/server.go`: Registered `PATCH /{id}/due-date` and `DELETE /{id}/due-date` routes

**SDK Client (Phase 5)**
- `internal/cli/sdk/client.go`: 
  - Updated `CreateTicket()` to accept `dueDate *time.Time`
  - Added `SetDueDate()` and `ClearDueDate()` methods
  - Updated `ListAllTickets()` and `ListTicketsByStatus()` to accept `dueBefore *time.Time`

**MCP Tools (Phase 6)**
- `internal/daemon/mcp/types.go`: Added `DueDate` to `CreateTicketInput`, `DueBefore` to `ListTicketsInput`, plus `UpdateDueDateInput`, `ClearDueDateInput` and output types
- `internal/daemon/mcp/tools_architect.go`: Added `updateDueDate` and `clearDueDate` tools with handlers, updated `handleCreateTicket` and `handleListTickets`

**Prompt Updates (Phase 7)**
- `internal/prompt/template.go`: Added `CurrentDate string` to `ArchitectKickoffVars`
- `internal/core/spawn/spawn.go`: Set `CurrentDate` in architect prompt building
- `internal/install/defaults/claude-code/prompts/architect/KICKOFF.md`: Added `**Current date**: {{.CurrentDate}}`

**TUI Updates (Phase 8)**
- `internal/cli/tui/kanban/styles.go`: Added `dueSoonStyle` and `overdueStyle`
- `internal/cli/tui/kanban/column.go`: Added due date indicators (`[OVERDUE]`, `[DUE SOON]`) to ticket cards
- `internal/cli/tui/ticket/styles.go`: Added matching due date styles
- `internal/cli/tui/ticket/model.go`: Added due date display in attributes panel with color-coding

**Test Files Updated**
- `internal/ticket/store_test.go`
- `internal/daemon/mcp/tools_test.go`
- `internal/notifications/dispatcher_test.go`

### Verification
- `make build` - ✅ Passed
- `make lint` - ✅ Passed (0 issues)
- `make test` - ✅ All tests passed