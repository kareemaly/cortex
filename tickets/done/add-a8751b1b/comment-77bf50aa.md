---
id: 77bf50aa-8c1b-454e-90af-a2312cafbafd
author: claude
type: ticket_done
created: 2026-01-28T07:32:47.393331Z
---
## Summary

Added `addTicketComment` as an architect MCP tool, enabling the architect agent to add comments to any ticket by ID. Previously, only ticket agents could add comments (scoped to their own session ticket).

## Changes Made

### `internal/daemon/mcp/types.go`
- Added `ArchitectAddCommentInput` struct with fields: `id` (ticket ID), `type` (comment type), `title`, and `content`. Follows the same pattern as existing architect input types (`ReadTicketInput`, `UpdateTicketInput`, etc.).

### `internal/daemon/mcp/tools_architect.go`
- Registered `addTicketComment` tool in `registerArchitectTools()` with description listing valid comment types.
- Added `handleArchitectAddComment` handler method that validates `id` and `title` fields, then delegates to `s.sdkClient.AddComment()` — the same SDK call used by the ticket agent version, but taking the ticket ID from input rather than session context.

## Key Decisions

- **Separate input type**: Created `ArchitectAddCommentInput` (with an `id` field) rather than reusing `AddCommentInput` (which has no `id` field). This keeps the architect and ticket tool input types cleanly separated, matching the existing pattern where architect tools always take an explicit ticket ID.
- **No changes to ticket tools**: `tools_ticket.go` remains unchanged — the ticket agent's `addTicketComment` continues to use `s.session.TicketID` implicitly.
- **Same validation pattern**: Validates `id` and `title` as required, consistent with other architect tool handlers.

## Verification
- `make build` — compiles successfully
- `make test` — all unit tests pass
- `make lint` — 0 issues

## Files Modified
- `internal/daemon/mcp/types.go` (+8 lines)
- `internal/daemon/mcp/tools_architect.go` (+30 lines)