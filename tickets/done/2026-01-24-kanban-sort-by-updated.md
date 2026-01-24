# Kanban Sort by Updated Date

## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

Tickets in kanban columns have no consistent ordering.

## Requirements

- Sort tickets in each column by updated date (most recent first)

## Implementation

### Commits
- `f779f2f` feat: sort kanban tickets by updated date (most recent first)
- `dc01d2c` Merge branch 'ticket/2026-01-24-kanban-sort-by-updated'

### Key Files Changed
- `internal/daemon/api/types.go` - Added `Updated` field to `TicketSummary` struct and populated it in `filterSummaryList()`
- `internal/cli/sdk/client.go` - Added `Updated` field to SDK's `TicketSummary` struct
- `internal/daemon/api/tickets.go` - Added sorting logic using `slices.SortFunc()` in both `ListAll()` and `ListByStatus()` handlers

### Approach
Server-side sorting in the API handlers ensures all clients (kanban TUI, CLI, MCP) receive consistently ordered results without needing client-side sorting logic.
