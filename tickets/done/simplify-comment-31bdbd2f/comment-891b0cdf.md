---
id: 891b0cdf-c943-4dda-b9a1-441878ca794e
author: claude
type: ticket_done
created: 2026-01-28T10:00:29.377414Z
---
## Summary

Simplified the comment model from 8 types to 4, removed the Title field from comments, added an Action field for structured data, and unified review requests into comments. This refactoring touched 17 files across all layers of the application.

## Key Changes

### Comment Type Simplification (8 → 4)
- **Before**: `scope_change`, `decision`, `blocker`, `progress`, `question`, `rejection`, `general`, `ticket_done`
- **After**: `review_requested`, `done`, `blocker`, `comment`

### Comment Struct Changes
- Removed `Title string` field
- Added `Action *CommentAction` field with `CommentAction{Type, Args}` and `GitDiffArgs{RepoPath, Commit}` types

### Review Unification
- Removed `RequestedReviews []ReviewRequest` from Session struct
- Removed `ReviewRequest` struct entirely
- Reviews are now comments with `type=review_requested` and `action={type: "git_diff", args: {repo_path, commit?}}`

### MCP Tool Restructuring
- Replaced single `addTicketComment` tool (which took Type, Title, Content) with two focused tools:
  - `addComment` — takes Content only, hardcoded type `comment`
  - `addBlocker` — takes Content only, hardcoded type `blocker`
- Updated `requestReview`: removed Title, added optional Commit field
- Updated `concludeSession`: renamed FullReport to Content

## Key Decisions

1. **4 comment types chosen**: `review_requested` and `done` are lifecycle events, `blocker` is an alert, `comment` is the general-purpose type. The old granular types (decision, progress, question, etc.) provided little value and complicated the UI.
2. **Action field for structured data**: Rather than having separate data models for reviews, the generic `CommentAction{Type, Args}` pattern allows comments to carry structured payloads (e.g., git_diff with repo_path and commit).
3. **Two MCP tools instead of one**: Splitting into addComment + addBlocker simplifies the agent interface — agents don't need to remember type strings.

## Files Modified (17)

### Core Model
- `internal/ticket/ticket.go` — Comment types, struct changes, removed ReviewRequest
- `internal/ticket/store.go` — Updated AddComment signature, removed AddReviewRequest

### Serialization Layer
- `internal/types/response.go` — Updated CommentResponse, added CommentActionResponse
- `internal/types/convert.go` — Updated conversion functions

### HTTP Layer
- `internal/daemon/api/types.go` — Updated request/response types
- `internal/daemon/api/tickets.go` — Updated handlers

### Client Layer
- `internal/cli/sdk/client.go` — Updated SDK methods

### Agent Interface
- `internal/daemon/mcp/types.go` — New input/output types
- `internal/daemon/mcp/tools_ticket.go` — Split addTicketComment into addComment + addBlocker
- `internal/daemon/mcp/tools_architect.go` — Updated architect comment tool

### UI Layer
- `internal/cli/tui/ticket/model.go` — Reviews from filtered comments, helper methods
- `internal/cli/tui/ticket/styles.go` — Updated comment type colors

### Documentation
- `.cortex/prompts/ticket-system.md` — Updated tool names
- `.cortex/prompts/architect.md` — Updated comment types

### Tests
- `internal/ticket/store_test.go` — Updated for new signatures
- `internal/ticket/ticket_test.go` — Updated CommentDecision → CommentGeneral, removed Title
- `internal/daemon/mcp/tools_test.go` — Updated all tool handler tests

## Verification
- `make build` — compiles cleanly
- `make test` — all tests pass
- `make lint` — 0 issues
- Commit: `bbdd2e2`, pushed to `origin/ticket/restructure-prompts-config`