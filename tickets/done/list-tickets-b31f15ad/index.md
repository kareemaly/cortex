---
id: b31f15ad-f31a-47b2-ba97-0c05883e45f7
title: List Tickets Require Status
type: ""
created: 2026-01-24T09:41:17Z
updated: 2026-01-24T09:41:17Z
---
## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

The `listTickets` MCP tool has status as optional. Architect should explicitly choose which status to query.

## Requirements

- Make `status` parameter required in `listTickets` MCP tool

## Implementation

### Commits

- `8e034a3` feat: enforce required status parameter in listTickets MCP tool

### Key Files Changed

- `internal/daemon/mcp/types.go` - Removed `omitempty` from Status json tag, updated jsonschema description to indicate required
- `internal/daemon/mcp/tools_architect.go` - Updated tool description, added validation for empty/invalid status, removed ListAll() fallback branch
- `internal/daemon/mcp/tools_test.go` - Updated 3 existing tests to include status, added 2 new tests for missing/invalid status validation
- `internal/daemon/mcp/integration_test.go` - Updated 3 tests to include status, added 2 new tests for missing/invalid status validation

### Decisions

- MCP SDK validates required fields at schema level before handler runs, so missing status returns SDK validation error rather than handler error
- Handler also validates status is one of the 4 valid enum values (backlog/progress/review/done) to provide clear error messages for invalid values