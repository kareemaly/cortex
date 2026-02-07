---
id: b657b748-6a79-4418-9922-99753bd5b4a8
title: 'Fix Broken Spawn: Tmux Session Target Prefix-Matching Bug'
type: ""
created: 2026-01-27T10:49:04.741856Z
updated: 2026-01-27T10:57:50.298353Z
---
## Problem

Spawning a ticket session creates the tmux window in the WRONG session (the daemon's session "Main" instead of the project's "cortex" session), causing cascading failures:

1. **Wrong tmux session** — Window created in "Main" instead of "cortex"
2. **send-keys fails** — `send-keys -t cortex:1` fails because window 1 doesn't exist in "cortex"
3. **No split panes** — Everything after window creation fails

Error: `"spawn: failed to spawn agent in tmux: tmux command failed: send-keys: can't find window: 1"`

## Root Cause

`CreateWindow` in `internal/tmux/window.go` uses `new-window -a -t session` where `-t` takes a `target-window`. Tmux's target-window resolution checks **window name prefix matches in the current session BEFORE checking session names**.

The daemon process runs in the "Main" tmux session which has a window named "cortexd". When it targets `-t cortex`:
1. Tmux checks window names in current session (Main) → "cortex" prefix-matches "cortexd" → targets Main:cortexd
2. Never reaches session name check for "cortex" session

**Fix:** Use `-t cortex:` (trailing colon) which forces session-level resolution, bypassing window name matching.

Verified:
- `TMUX_PANE=%518 tmux new-window -a -t cortex ...` → `Main:1` (BUG)
- `TMUX_PANE=%518 tmux new-window -a -t cortex: ...` → `cortex:1` (FIXED)

## Fix Applied

Changed `CreateWindow` in `internal/tmux/window.go` to format session target as `fmt.Sprintf("%s:", session)`.

## Remaining Scope

- [x] Fix tmux session targeting in `CreateWindow`
- [ ] Rebuild daemon and verify spawn works
- [ ] Clean up worktree issue (separate from this fix — worktrees temporarily disabled)
- [ ] `make build && make test` pass