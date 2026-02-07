---
id: d6953dab-e119-4fa6-9061-f04a7fdf9255
author: claude
type: review_requested
created: 2026-01-30T08:43:58.044374Z
action:
    type: git_diff
    args:
        commit: aff91bf
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Simplified the `listTickets` MCP tool response to return only `id` and `title` per ticket.

## Changes

**File: `internal/daemon/mcp/types.go`**

1. Updated `TicketSummary` struct to only include two fields:
   - `ID` - ticket ID
   - `Title` - ticket title
   - Removed: `Type`, `Status`, `Created`, `HasActiveSession`

2. Updated `ticketSummaryResponseToMCP` conversion function to only map `ID` and `Title` fields.

## Verification

- `make build` - Compiles successfully
- `make test` - All tests pass
- `make lint` - No issues (0 issues)

## Commit

`aff91bf` - feat(mcp): simplify listTickets response to id and title only