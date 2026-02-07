---
id: 37268178-5c2c-481b-88b7-0b0de46f9453
title: 'Kanban: Add "gd" Shortcut to Focus Daemon Window, Remove "a" Approve Shortcut'
type: ""
created: 2026-01-27T13:22:00.586117Z
updated: 2026-01-27T13:50:12.225145Z
---
## Summary

Two keybinding changes to the kanban TUI.

## Changes

### 1. Add "gd" shortcut to focus CortexDaemon window 0
- Pressing `gd` in the kanban view should focus tmux window 0 of the `CortexDaemon` session
- This provides quick access to the daemon dashboard from the kanban board
- Should go through the daemon HTTP API for the tmux focus operation

### 2. Remove "a" (approve) shortcut from kanban
- The approve action should only be available from inside a ticket detail view, not from the kanban board
- Remove the `a` keybinding and its handler from the kanban TUI
- Remove it from the help bar if shown there

## Files
- `internal/cli/tui/kanban/` — kanban TUI model, keys, view