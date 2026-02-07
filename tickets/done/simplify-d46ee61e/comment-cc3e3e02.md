---
id: cc3e3e02-d4e3-482f-8e49-694cd59afe6c
author: claude
type: done
created: 2026-01-30T08:46:46.031543Z
---
## Summary

Simplified the `listTickets` MCP tool response to return only `id` and `title` per ticket, reducing context usage for architect agents.

## Changes Made

### File: `internal/daemon/mcp/types.go`

1. **Simplified `TicketSummary` struct** (lines 110-113)
   - Before: 6 fields (`ID`, `Type`, `Title`, `Status`, `Created`, `HasActiveSession`)
   - After: 2 fields (`ID`, `Title`)

2. **Updated `ticketSummaryResponseToMCP` conversion function** (lines 210-216)
   - Simplified to only map `ID` and `Title` fields from the shared types

## Rationale

- Architects using `listTickets` primarily need to identify tickets by ID and title
- Full details (type, status, dates, session info) are available via `readTicket` when needed
- Minimal response payload reduces context window usage for LLM-based architects

## Verification

- `make build` - Compiles successfully
- `make test` - All unit tests pass
- `make lint` - No issues

## Commit

- `aff91bf` - feat(mcp): simplify listTickets response to id and title only
- Pushed to `origin/main`

## Files Modified

| File | Lines Changed |
|------|---------------|
| `internal/daemon/mcp/types.go` | 7 insertions, 15 deletions |