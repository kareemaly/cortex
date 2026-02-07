---
id: 4391eb80-c1e8-4fa3-8630-7134b8b09cb9
title: 'Ticket TUI: Add "ga" Shortcut to Focus Architect Window'
type: ""
created: 2026-01-27T14:07:40.772235Z
updated: 2026-01-27T14:44:43.152723Z
---
## Summary

Add a `ga` (go architect) two-key shortcut to the ticket detail TUI that focuses the architect window (window 0) of the current tmux session.

## Changes

- Add `ga` key sequence to the ticket detail TUI (follows same `pendingG` pattern as kanban's `gd`)
- On `ga`, call the daemon API to focus tmux window 0 of the current project's tmux session (the architect window)
- The focus operation needs both `select-window` and `switch-client` (lesson from the `gd` fix — both are required for cross-session switching)

## Implementation Notes

- The ticket TUI runs inside a ticket agent's tmux window. The architect is window 0 of the same tmux session (the project session, e.g., "cortex")
- May need a new daemon API endpoint like `POST /architect/focus` or reuse existing focus infrastructure
- The project's tmux session name comes from the project config (`name` field in `.cortex/cortex.yaml`)

## Files
- `internal/cli/tui/ticket/` — ticket detail TUI model, keys
- `internal/daemon/api/` — may need a focus endpoint for architect window
- `internal/cli/sdk/client.go` — SDK method for the focus call