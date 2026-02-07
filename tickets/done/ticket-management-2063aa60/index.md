---
id: 2063aa60-f76b-4191-bdd4-92715a02d4ec
title: Ticket Management
type: ""
created: 2026-01-20T08:39:06Z
updated: 2026-01-20T08:39:06Z
---
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

## Implementation

### Commits

- `f9ec5a5` feat: add ticket management package with JSON-based storage

### Key Files

| File | Purpose |
|------|---------|
| `internal/ticket/errors.go` | `NotFoundError`, `ValidationError` types and `IsNotFound()` helper |
| `internal/ticket/slug.go` | `GenerateSlug()` - URL-friendly slugs (max 20 chars, word-boundary truncation) |
| `internal/ticket/ticket.go` | Data structures: `Ticket`, `Session`, `Report`, `StatusEntry`, `Dates` |
| `internal/ticket/store.go` | `Store` with CRUD + session management, file-based JSON storage |
| `internal/ticket/*_test.go` | 24 tests covering all operations |

### Decisions

- Used `github.com/google/uuid` for ID generation
- File naming: `{slug}-{shortID}.json` where shortID is first 8 chars of UUID
- Status directories created on `NewStore()` initialization
- `dates.approved` set automatically when moving ticket to done
- Session status history appends on each update (full audit trail)