---
id: 851f4b57-2d53-48dd-9e85-104f515a9538
title: Add due date support for tickets
type: work
created: 2026-02-05T07:50:47.957944Z
updated: 2026-02-05T08:09:04.256491Z
---
## Objective

Add optional due date field to tickets with MCP tools for management, visual indicators in TUI, and injection into architect prompts.

## Data Model

Update `internal/ticket/ticket.go`:
```go
type Ticket struct {
    // ...existing fields
    DueDate *time.Time `json:"due_date,omitempty"`
}
```

## MCP Tools (Architect)

### 1. Update `createTicket`
- Add optional `dueDate` parameter (ISO 8601 string)
- Parse and store as UTC

### 2. New tool: `updateDueDate`
- Parameters: `id` (required), `dueDate` (required, ISO 8601)
- Updates due date for specified ticket

### 3. New tool: `clearDueDate`
- Parameters: `id` (required)
- Removes due date from ticket (sets to null)

### 4. Update `listTickets`
- Add optional `dueBefore` filter parameter (ISO 8601)
- Returns tickets with due date before specified time

### Types location: `internal/daemon/mcp/types.go`
### Tools location: `internal/daemon/mcp/tools_architect.go`

## API Layer

Update HTTP handlers in `internal/daemon/api/tickets.go`:
- Create endpoint accepts `due_date`
- Add `PATCH /tickets/{id}/due-date` endpoint
- Add `DELETE /tickets/{id}/due-date` endpoint
- List endpoint accepts `due_before` query param

Update SDK client `internal/cli/sdk/client.go` accordingly.

## Architect Kickoff Prompt

Update `internal/install/defaults/claude-code/prompts/architect/KICKOFF.md`:
- Inject current date: `**Current date**: {{.CurrentDate}}`
- Show due dates in ticket list with warning indicator for urgent items

Update template vars in `internal/prompt/template.go` to include `CurrentDate`.

## TUI Updates

### Kanban (`internal/cli/tui/kanban/`)

Update card rendering in `column.go`:
- Due within 24h: show `⚠️` or red styling
- Overdue: show `🔴` or stronger highlight  
- Has due date (not urgent): show subtle date `(Feb 10)`

Add styles in `styles.go` for due date indicators.

Consider: sort tickets by due date within columns (soonest first).

### Ticket Detail (`internal/cli/tui/ticket/`)
- Display due date in ticket header if set

## Acceptance Criteria

- [ ] Ticket struct has optional `DueDate` field
- [ ] `createTicket` accepts optional `dueDate` param
- [ ] `updateDueDate` tool works
- [ ] `clearDueDate` tool works  
- [ ] `listTickets` supports `dueBefore` filter
- [ ] Architect kickoff shows current date
- [ ] Architect kickoff shows due dates in ticket list
- [ ] Kanban shows due date indicators (⚠️ for <24h, 🔴 for overdue)
- [ ] `make build && make lint && make test` pass