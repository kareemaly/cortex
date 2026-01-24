# Consolidate Response Types

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
