---
id: 7ab7af63-f87d-4345-ab06-c445d379860d
title: Display due dates in architect injection and TUIs
type: debug
tags:
    - tui
    - mcp
    - due-date
created: 2026-02-07T11:18:13.537659Z
updated: 2026-02-07T11:23:41.648718Z
---
## Problem

Due dates are not surfaced in two places:

1. **Architect context injection** — When tickets are injected into the architect system prompt, the due date field is omitted. Architects cannot see or reason about deadlines.
2. **Kanban TUI** (`cortex kanban`) — Due dates are not displayed on ticket cards/rows.
3. **Ticket detail TUI** (`cortex show`) — Due date is not shown in the ticket detail view.

## Acceptance Criteria

- [ ] Architect system prompt injection includes due date for each ticket that has one (format: human-readable, e.g. `(due: 2026-02-10)`)
- [ ] Kanban TUI displays due date on tickets that have one
- [ ] Ticket detail TUI (`cortex show`) displays due date
- [ ] Overdue tickets should be visually distinguishable (e.g. red/warning color) in TUIs

## Key Paths to Investigate

- Architect context injection: likely in `internal/daemon/mcp/` (architect tools or prompt construction)
- Kanban TUI: `internal/cli/tui/`
- Ticket detail TUI: `internal/cli/tui/`
- Ticket response types: `internal/types/`