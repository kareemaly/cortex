---
id: e9969910-88d5-4ab9-8710-223d46dd3fee
author: claude
type: review_requested
created: 2026-02-07T09:57:26.798604Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Updated all MCP tools to work with the new frontmatter + directory-per-entity storage layer. This wires new fields, adds new tools, and updates the full stack (store → HTTP API → SDK → MCP).

## Changes by Layer

### Ticket Store (`internal/ticket/store.go`)
- `Create()` now accepts `tags []string` parameter
- `Update()` now accepts `tags *[]string` parameter

### Shared Types (`internal/types/`)
- `DocResponse` now includes `Comments []CommentResponse`
- `ToDocResponse()` maps doc comments to the response

### HTTP API (`internal/daemon/api/`)
- **types.go**: Added `Tags` to `CreateTicketRequest` and `UpdateTicketRequest`, `Author` to `AddCommentRequest`, new `AddDocCommentRequest` type, `tag` filter parameter to `filterSummaryList()`
- **tickets.go**: Pass tags through to store Create/Update, use explicit author if provided in AddComment, read `tag` query param for list filtering
- **docs.go**: New `AddComment` handler for adding comments to docs
- **sessions.go**: New `List` handler returning active sessions with resolved ticket titles
- **server.go**: Registered `POST /docs/{id}/comments` and `GET /sessions/` routes

### SDK Client (`internal/cli/sdk/client.go`)
- `CreateTicket()`: Added `tags []string` parameter
- `UpdateTicket()`: Added `tags *[]string` parameter
- `ListAllTickets()`: Added `tag string` parameter
- `ListTicketsByStatus()`: Added `tag string` parameter
- `AddComment()`: Added `author string` parameter
- New `AddDocComment()` method
- New `ListSessions()` method

### MCP Types (`internal/daemon/mcp/types.go`)
- `TicketSummary` enriched with `Type`, `Tags`, `Due`, `Created`, `Updated` fields
- `DocOutput` now includes `Comments []CommentOutput`
- `ListTicketsInput` has new `Tag` field
- `CreateTicketInput` has new `Tags` field
- `UpdateTicketInput` has new `Tags` field
- New types: `AddDocCommentInput`, `ListSessionsInput`, `SessionListItem`, `ListSessionsOutput`
- Updated conversion functions to map all new fields

### MCP Handlers (`internal/daemon/mcp/tools_architect.go`, `tools_ticket.go`)
- `handleListTickets`: passes `input.Tag` to SDK
- `handleCreateTicket`: passes `input.Tags` to SDK
- `handleUpdateTicket`: passes `input.Tags` to SDK
- `handleArchitectAddComment`: passes `"architect"` as author
- `handleAddComment`/`handleAddBlocker` (ticket tools): passes empty author (API resolves from session)
- New `handleAddDocComment` handler — validates, calls `AddDocComment` with author "architect"
- New `handleListSessions` handler — calls `ListSessions`, maps to MCP output
- Both new tools registered in `registerArchitectTools()`

### Tests
- Updated all `store.Create()` calls across 4 test files for new signature
- Updated all `store.Update()` calls for new signature
- All existing tests pass

### Verification
- `make build` ✅
- `make lint` ✅ (0 issues)
- `make test` ✅ (all pass)