# Kanban Ticket Detail Navigation

## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

From the kanban board, there's no way to view ticket details without leaving the TUI.

## Requirements

- Press `o` or `enter` on a ticket in kanban to open the ticket detail view
- Press `q` or `esc` in detail view to return to kanban (not quit the app)
- Reuse existing `internal/cli/tui/ticket` components for the detail view
