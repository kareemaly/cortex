---
id: 172522e7-094b-4ca6-bcab-171798bd8d74
title: Consolidate Response Types
type: ""
created: 2026-01-24T15:58:09Z
updated: 2026-01-24T15:58:09Z
---
## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

Same response types are defined 3 times across different packages:

- `internal/cli/sdk/client.go` - SDK response types
- `internal/daemon/api/types.go` - API response types
- `internal/daemon/mcp/types.go` - MCP output types

Duplicated types include:
- `TicketSummary` (3 copies)
- `DatesResponse/DatesOutput` (3 copies)
- `CommentResponse/CommentOutput` (3 copies)
- `SessionResponse/SessionOutput` (3 copies)
- `StatusEntryResponse/StatusOutput` (3 copies)

Also, conversion functions `toTicketResponse()` and `ToTicketOutput()` do nearly identical work.

## Requirements

- Create shared types package (e.g., `internal/types/`)
- Define canonical response types once
- Update SDK, API, and MCP packages to use shared types
- Consolidate conversion functions

## Implementation

### Commits

- `b1a14fe` refactor: consolidate response types into internal/types package

### Key Files Changed

**Created:**
- `internal/types/response.go` - 13 shared response types
- `internal/types/convert.go` - 7 shared conversion helpers

**Modified:**
- `internal/daemon/api/types.go` - Type aliases for shared types, kept request types + SpawnResponse
- `internal/cli/sdk/client.go` - Type aliases for shared types, kept HealthResponse + APIError
- `internal/daemon/mcp/types.go` - Type aliases for identical types, kept MCP-specific types
- `internal/daemon/api/tickets.go` - Updated to use `types.ToTicketResponse()` and `types.ToSessionResponse()`

### Decisions

1. **Type aliases over imports** - Used Go type aliases (`type X = types.X`) in api and sdk packages to maintain backwards compatibility for consumers using `sdk.TicketSummary` etc.

2. **MCP keeps some unique types** - MCP's `TicketSummary`, `SessionOutput`, and `TicketOutput` have structural differences (e.g., MCP has `IsActive` but no `StatusHistory`), so kept as separate types while using shared conversion helpers for common fields.

3. **Shared conversion helpers** - Created `types.ToCommentResponse()`, `types.ToDatesResponse()`, etc. that MCP conversion functions now call internally.

### Scope Changes

None - implemented as specified.