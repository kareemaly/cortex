---
id: d618aab7-9e85-4dbe-aa5d-807889b55e81
author: claude
type: ticket_done
created: 2026-01-27T10:28:56.138568Z
---
## Summary

Fixed the architect agent pane width issue where re-spawning the architect on a pre-existing tmux session caused incorrect pane sizing. The architect pane now consistently renders at 30% width, matching the ticket agent layout.

## Root Cause

When `SpawnArchitect` ran on a pre-existing session, window 0 could already have stale panes from a previous architect spawn (agent pane + companion pane). The function never cleaned up these panes before splitting:

1. Session already exists → window 0 has 2 panes (old agent + old companion)
2. `CreateArchitectWindow` just renames window 0 — no pane cleanup
3. `SplitWindowHorizontal` splits the active pane again → 3 panes with wrong sizing

By contrast, `SpawnAgent` always creates a fresh window via `CreateWindow`, so it always starts clean.

## Changes

### `internal/tmux/pane.go`
- Added `ResetWindowPanes(session, windowIndex)` method that runs `tmux kill-pane -a -t <session>:<window>.0` to kill all panes except pane 0, restoring a single-pane layout. Errors are intentionally discarded (no-op when already a single pane).

### `internal/tmux/command.go`
- Added a `ResetWindowPanes` call in `SpawnArchitect` after `CreateArchitectWindow` and before executing the agent command. This ensures stale panes from a previous spawn are cleaned up before the 70/30 split.

## Key Decisions

- **Ignore errors from `kill-pane -a`**: When window 0 already has only one pane, tmux returns an error. This is expected and harmless, so the error is explicitly discarded with `_, _ = m.run(...)` to satisfy the linter.
- **Minimal change scope**: Only added the pane reset step rather than restructuring `SpawnArchitect` to create fresh windows, preserving the existing architect-uses-window-0 design.

## Verification

- `make build` — passes
- `make lint` — 0 issues
- `make test` — all unit tests pass
- `make test-integration` — MCP integration tests have pre-existing connection failures unrelated to this change; API integration tests pass