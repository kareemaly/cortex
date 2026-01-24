# Kanban Ticket Card Layout

## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

Ticket titles in kanban columns are truncated. No visibility into when tickets were created.

## Requirements

- Word wrap ticket titles across multiple lines (no truncation)
- Display creation date in faded text underneath the title

## Implementation

### Commits
- `fd37cd1` feat: kanban ticket card word wrap and creation date display
- `fa4f9ed` Merge branch 'ticket/2026-01-23-kanban-ticket-card-layout'
- `383762f` fix: remove title truncation in kanban ticket cards, allow full word wrap
- `c590f7b` Merge branch 'ticket/2026-01-23-kanban-ticket-card-layout'

### Key Files Changed
- `internal/cli/tui/kanban/column.go` - Added word wrapping, multi-line card rendering, creation date display, removed truncation

### Decisions
- Titles wrap fully with no line limit or `...` truncation
- Each ticket card takes up to 6 lines (5 for title + 1 for date)
- Date format: `Jan 2` (e.g., "Jan 23")
- Selected tickets show date with highlight style, unselected show muted style

### Scope Changes
- Initial implementation had 2-line limit with truncation; follow-up removed this per user feedback
