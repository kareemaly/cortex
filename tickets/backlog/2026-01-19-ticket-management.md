# Ticket Management

Implement the ticket file management layer for JSON-based ticket storage.

## Context

Tickets are JSON files stored in `.cortex/tickets/{backlog,progress,done}/`. This package provides CRUD operations and is the foundation for MCP tools and daemon API.

See `DESIGN.md` for:
- Ticket JSON schema (line 81-150)
- Ticket lifecycle (line 74-79)
- Status values (line 162-168)

## Requirements

Create `internal/ticket/` package that supports:

1. **Data structures** matching the JSON schema in DESIGN.md
2. **CRUD operations**: Create, Read, Update, Delete tickets
3. **List tickets** by status (backlog, progress, done)
4. **Move tickets** between statuses
5. **Session management** within tickets (add session, get active session)
6. **Slug generation** from title (max 20 chars, per DESIGN.md line 154-159)

## Verification

```bash
make build   # Builds successfully
make test    # Tests pass
make lint    # No lint errors
```

## Notes

- New tickets go to backlog
- Moving to done should set `dates.approved`
- Store should search all status folders when reading by ID
- Include unit tests for ticket operations and store
