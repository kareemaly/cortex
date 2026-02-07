---
id: edfa4f14-d529-40f2-8318-6cb42ad4cdc8
title: Add Title Field to Comments and Reviews, Rename Review Summary to Content
type: ""
created: 2026-01-27T15:19:30.266338Z
updated: 2026-01-27T16:34:08.404464Z
---
## Summary

Normalize the data model for comments and reviews so both have a required `Title` + `Content` pattern. Currently comments have `Type` + `Content` (no title), and reviews have `Summary` (which is actually full markdown content, not a title).

## Changes

### Comments: Add required `Title` field
- Add `Title string` to the `Comment` struct
- Title is a short one-liner summary, `Content` remains the full markdown body
- Both `Title` and `Content` are required

### Reviews: Add required `Title`, rename `Summary` → `Content`
- Add `Title string` to `RequestedReview` struct
- Rename `Summary` → `Content` (it contains full markdown, not a summary)
- Both `Title` and `Content` are required

### Enforce required fields everywhere

Audit and update ALL of the following to require both `title` and `content`:

**Schema:**
- `internal/ticket/ticket.go` — `Comment` struct, `RequestedReview` struct

**Store:**
- `internal/ticket/store.go` — `AddComment`, `RequestReview` methods — validate both fields are non-empty

**API Types:**
- `internal/types/response.go` — `CommentResponse`, `RequestedReviewResponse` — add Title, rename Summary→Content

**HTTP API handlers:**
- `internal/daemon/api/tickets.go` — `AddComment` handler, `RequestReview` handler — require title+content in request body, return validation error if missing

**MCP Tools:**
- `internal/daemon/mcp/tools_ticket.go` — `addTicketComment` tool, `requestReview` tool — add title input parameter, require both fields
- Update MCP tool schemas/descriptions

**SDK Client:**
- `internal/cli/sdk/client.go` — `AddComment`, `RequestReview` methods — accept title parameter

**Tests:**
- Update all tests that create comments or reviews to include title
- Add validation tests for missing title/content

## Notes

- This is a breaking change to the review schema (Summary→Content rename). Existing review JSON files will need the field renamed.
- Existing comments without titles: handle gracefully during migration (treat as empty string or derive from first line of content)

## Files
- `internal/ticket/ticket.go` — Comment, RequestedReview structs
- `internal/ticket/store.go` — AddComment, RequestReview methods
- `internal/types/response.go` — CommentResponse, RequestedReviewResponse
- `internal/daemon/api/tickets.go` — AddComment, RequestReview handlers
- `internal/daemon/mcp/tools_ticket.go` — addTicketComment, requestReview tools
- `internal/cli/sdk/client.go` — SDK methods
- `internal/daemon/mcp/tools_test.go` — MCP tests
- `internal/daemon/api/integration_test.go` — API tests