---
id: 04639105-a7b6-40d6-9f8f-52593800ca0b
title: 'Fix Dashboard Focus: Investigate Broken Focusing and Route Through Daemon API'
type: ""
created: 2026-01-27T13:00:29.161951Z
updated: 2026-01-27T13:42:12.890464Z
---
## Summary

Focusing a session from the dashboard TUI is currently broken. Investigate the root cause and ensure all focus operations go through the daemon HTTP API rather than direct tmux calls from the client.

## Problems

1. Focusing a session from the dashboard does not work correctly — needs investigation into what exactly fails (wrong tmux target, session not found, race condition, etc.)
2. The focus path may be calling tmux directly from the CLI client rather than going through the daemon API, which violates the architecture principle that all clients communicate exclusively over HTTP

## Investigation

- Trace the focus flow from dashboard keypress (`F` key / `Enter` on session row) through to the actual tmux operation
- Check `spawnArchitect()` and `focusTicket()` in the dashboard model
- Check the SDK client methods these call
- Verify whether the daemon API has proper focus/attach endpoints or if the client is shelling out to tmux directly
- If no daemon endpoint exists for focusing, add one

## Expected Outcome

- Focusing a session from the dashboard reliably switches to the correct tmux window
- All focus operations route through `cortexd` HTTP API — no direct tmux calls from the CLI/TUI client

## Files to investigate
- `internal/cli/tui/dashboard/model.go` — focus command handlers
- `internal/cli/sdk/client.go` — SDK client methods
- `internal/daemon/api/` — check for focus/attach endpoints
- `internal/tmux/` — tmux manager used by daemon