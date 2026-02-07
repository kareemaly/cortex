---
id: 43b4c7a0-2f51-4c70-bc01-43311ef07a63
title: Session End via MCP Restructure
type: ""
created: 2026-01-23T13:50:50Z
updated: 2026-01-23T13:50:50Z
---
## Context

Early development, no users, breaking changes acceptable, no tech debt.

## Problem

- `concludeSession` MCP tool shouldn't exist - agents shouldn't end their own sessions
- `moveTicketToDone` needs to capture completion summary

## Changes

- Remove `concludeSession` from ticket MCP tools
- Add `ticket_done` comment type
- Update `moveTicketToDone` to accept optional summary, add as `ticket_done` comment
- Remove deprecated MCP tools (`pickupTicket`, `submitReport`, `approve`)

## Verification

```bash
make lint
make test
make build
make test-integration
```

## Implementation

### Commits
- `b1818d4` refactor: remove deprecated MCP tools, add summary support to moveTicketToDone

### Key Files Changed
- `internal/ticket/ticket.go` - Added `CommentTicketDone` type
- `internal/daemon/mcp/types.go` - Added `MoveTicketToDoneInput`, removed deprecated input/output types
- `internal/daemon/mcp/tools_ticket.go` - Updated `moveTicketToDone` handler, removed deprecated tools
- `internal/daemon/mcp/server.go` - Removed unused `killSessionWindow` function
- `internal/daemon/mcp/tools_test.go` - Updated tests, added `TestHandleMoveTicketToDone_WithSummary`

### Decisions
- Summary provided to `moveTicketToDone` is stored as `ticket_done` comment type (not `progress`)
- Removed all deprecated tools in single commit since no backward compatibility needed
- Cleaned up unused `killSessionWindow` function that was only used by removed handlers