---
id: 42179c34-9238-457a-8566-960b1a95746e
author: claude
type: ticket_done
created: 2026-01-27T16:34:08.40313Z
---
## Summary

Added a required `Title` field to both `Comment` and `ReviewRequest` structs, and renamed `ReviewRequest.Summary` to `Content`. This normalizes the data model so both comments and reviews follow a consistent `Title` + `Content` pattern, where Title is a short one-liner and Content is the full markdown body.

## Changes Made

### Core Structs (`internal/ticket/ticket.go`)
- Added `Title string` field to `Comment` struct (after `Type`)
- Added `Title string` field to `ReviewRequest` struct (after `RepoPath`)
- Renamed `ReviewRequest.Summary` → `Content` with updated json tag

### Store Methods (`internal/ticket/store.go`)
- `AddComment`: Added `title` parameter, added non-empty validation for both `title` and `content`
- `AddReviewRequest`: Added `title` parameter, renamed `summary` → `content`, added non-empty validation for both

### Response Types (`internal/types/response.go`)
- Added `Title` to `CommentResponse` and `RequestedReviewResponse`
- Renamed `Summary` → `Content` in `RequestedReviewResponse`

### Type Conversions (`internal/types/convert.go`)
- Updated `ToCommentResponse` to map `Title`
- Updated `ToRequestedReviewResponse` to map `Title` and `Content` (was `Summary`)

### API Types (`internal/daemon/api/types.go`)
- Added `Title` to `AddCommentRequest`
- Added `Title` and renamed `Summary` → `Content` in `RequestReviewRequest`

### API Handlers (`internal/daemon/api/tickets.go`)
- `AddComment` handler: Added title validation (400 if empty), passes title to store
- `RequestReview` handler: Added title validation, renamed summary → content validation and passing
- `Conclude` handler: Passes "Session concluded" as title when creating ticket_done comment

### MCP Types (`internal/daemon/mcp/types.go`)
- Added `Title` to `AddCommentInput` and `RequestReviewInput` with jsonschema descriptions
- Renamed `Summary` → `Content` in `RequestReviewInput`

### MCP Tool Handlers (`internal/daemon/mcp/tools_ticket.go`)
- `handleAddTicketComment`: Added title validation, passes title to SDK client
- `handleRequestReview`: Added title/content validation, passes both to SDK client

### SDK Client (`internal/cli/sdk/client.go`)
- `AddComment`: Added `title` parameter, includes in request body
- `RequestReview`: Added `title` parameter, renamed `summary` → `content` in request body

### TUI (`internal/cli/tui/ticket/model.go`)
- Review requests: Display `review.Title` instead of `review.Summary`
- Comments: Show `comment.Title` on the badge line after type badge and before date

### Tests
- `ticket_test.go`: Added `Title` to Comment in JSON round-trip test
- `store_test.go`: Updated existing AddComment tests with title param, added `TestStoreAddCommentValidation`, `TestStoreAddReviewRequest`, `TestStoreAddReviewRequestValidation`
- `tools_test.go`: Updated existing comment tests with Title field, added `TestHandleAddTicketCommentMissingTitle`, `TestHandleRequestReview`, `TestHandleRequestReviewValidation`

## Key Decisions
- **Title required everywhere**: Both title and content are required (non-empty) at the store level and MCP handler level, enforced with ValidationError
- **Conclude uses fixed title**: The Conclude handler passes "Session concluded" as the title for the auto-generated ticket_done comment
- **TUI displays title prominently**: Review requests show title as the primary text; comments show title inline on the badge line

## Verification
- `make build` — compilation succeeds
- `make test` — all 11 test suites pass (including 6 new test cases)
- `make lint` — 0 issues

## Files Modified (13)
1. `internal/ticket/ticket.go`
2. `internal/ticket/store.go`
3. `internal/ticket/ticket_test.go`
4. `internal/ticket/store_test.go`
5. `internal/types/response.go`
6. `internal/types/convert.go`
7. `internal/daemon/api/types.go`
8. `internal/daemon/api/tickets.go`
9. `internal/daemon/mcp/types.go`
10. `internal/daemon/mcp/tools_ticket.go`
11. `internal/daemon/mcp/tools_test.go`
12. `internal/cli/sdk/client.go`
13. `internal/cli/tui/ticket/model.go`

## Follow-up Notes
- This is a breaking change to the JSON schema (Summary → Content rename). Existing review JSON files with the old `summary` field will need migration or will have empty Content on deserialization.