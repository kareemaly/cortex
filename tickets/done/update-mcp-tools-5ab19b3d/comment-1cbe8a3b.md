---
id: 1cbe8a3b-620e-4819-b59f-a356e38ccd6d
author: claude
type: done
created: 2026-02-07T09:59:25.813736Z
---
## Work Done

Updated all MCP tools to properly work with the new frontmatter + directory-per-entity storage layer, wiring new fields through the full stack (store → HTTP API → SDK → MCP).

### Changes (20 files, +511/-137 lines)

**Store layer:** Added `tags` parameter to `ticket.Store.Create()` and `ticket.Store.Update()`.

**Shared types:** Added `Comments` to `DocResponse`, updated `ToDocResponse()` to map doc comments.

**HTTP API:** Added `Tags` to create/update ticket requests, `Author` to `AddCommentRequest`, tag filter to list endpoints, new `POST /docs/{id}/comments` and `GET /sessions/` endpoints.

**SDK client:** Updated `CreateTicket`, `UpdateTicket`, `ListAllTickets`, `ListTicketsByStatus`, `AddComment` signatures. Added `AddDocComment()` and `ListSessions()` methods.

**MCP types:** Enriched `TicketSummary` with type/tags/due/created/updated. Added comments to `DocOutput`. New input/output types for `addDocComment` and `listSessions`.

**MCP handlers:** Updated all existing handlers for new signatures. Added `handleAddDocComment` and `handleListSessions`. Architect tools set author to "architect", ticket tools let API resolve from session.

**Tests:** Updated all `store.Create()`/`store.Update()` calls across 4 test files.

### Verification
- `make build` ✅
- `make lint` ✅ (0 issues)  
- `make test` ✅ (all pass)

### Commit
`1a3c6f2` on `feat/frontmatter-storage`