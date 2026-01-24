# Kanban Ticket Detail Navigation

## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

From the kanban board, there's no way to view ticket details without leaving the TUI.

## Requirements

- Press `o` or `enter` on a ticket in kanban to open the ticket detail view
- Press `q` or `esc` in detail view to return to kanban (not quit the app)
- Reuse existing `internal/cli/tui/ticket` components for the detail view

## Implementation

### Commits
- `a86299a` feat: add ticket detail navigation from kanban with o/enter and q/esc to return

### Key Files Changed
- `internal/cli/tui/ticket/model.go` - Added `CloseDetailMsg`, `embedded` field, `NewEmbedded()` constructor
- `internal/cli/tui/ticket/keys.go` - Updated `helpText()` to show "[q/esc] back" when embedded
- `internal/cli/tui/kanban/keys.go` - Added `KeyOpen` constant, updated help text
- `internal/cli/tui/kanban/model.go` - Added detail view delegation in Update/View, open handler for o/enter

### Decisions
- Used model composition: kanban embeds `*ticket.Model` and delegates Update/View when showing detail
- Added `embedded` mode to ticket model that sends `CloseDetailMsg` instead of `tea.Quit`
- Refresh tickets on return from detail view to pick up any changes
