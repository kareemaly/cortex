---
id: d46ee61e-83d2-4111-a94a-ff58bd668c91
title: Simplify listTickets Response to ID and Title Only
type: work
created: 2026-01-29T16:05:51.35484Z
updated: 2026-01-30T08:46:46.032536Z
---
## Summary

Reduce the `listTickets` MCP tool response to return only ticket ID and title. This keeps the list view minimal and reduces context usage for architects.

## Requirements

- `listTickets` should return an array of objects with only:
  - `id` — ticket ID
  - `title` — ticket title
- Remove other fields from list response (body, status, dates, comments, session, etc.)
- Architects can use `readTicket` to get full details when needed

## Acceptance Criteria

- [ ] `listTickets` returns only `id` and `title` per ticket
- [ ] Response payload is minimal
- [ ] `readTicket` still returns full ticket details (unchanged)