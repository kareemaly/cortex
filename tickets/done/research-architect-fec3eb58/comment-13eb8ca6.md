---
id: 13eb8ca6-dad2-4835-90ca-611f2a9a7700
author: claude
type: done
created: 2026-02-13T10:20:01.451078Z
---
## Research: Architect Tmux Pane Split 50/50 Instead of 30/70

### Root Cause Identified
The 50/50 split occurs because tmux detached sessions default to 80x24 (`default-size`). When `SpawnArchitect` creates a new detached session and splits panes with `-p 70`, the split is correct at 80 columns (24/56). However, when a client attaches and the window resizes to the actual terminal width, **tmux distributes extra columns evenly across panes** (not proportionally), causing the ratio to drift toward 50/50.

Ticket agents are unaffected because they create new windows in an already-attached session, so the split happens at the correct terminal width.

### Fixes Attempted (Both Failed)
1. **Option A - Thread terminal dimensions**: Passed terminal width/height from CLI through SDK → API → spawn → tmux to set `default-size` before session creation. Failed because the dashboard TUI spawn path doesn't have terminal dimensions available (passes 0,0).

2. **Option B - Tmux one-shot hooks**: Added `window-resized[99]` and `client-session-changed[99]` hooks to fire `resize-pane -x 70%` on first client attach, then self-remove. Empirically verified in isolation but did not work in practice when spawning from the dashboard.

### Remaining Investigation Needed
- The hooks approach was verified to work in manual tmux testing but failed in the actual dashboard spawn flow. Possible causes:
  - Hook timing: the hooks may fire before the window is fully visible
  - The dashboard's `SwitchClient` path may not trigger these specific hooks
  - tmux hook execution order or event suppression during rapid session setup
- A `after-client-attached` or `session-created` hook might be more appropriate
- Alternative: set `default-size` on the session option directly via `tmux set-option -t session default-size WIDTHxHEIGHT` if dimensions can be obtained server-side

### Files Investigated
- `internal/tmux/command.go` - SpawnArchitect, SpawnAgent
- `internal/tmux/session.go` - CreateSession (detached, 80x24 default)
- `internal/tmux/pane.go` - SplitWindowHorizontal (hardcoded -p 70)
- `internal/core/spawn/spawn.go` - spawnInTmux dispatch
- `internal/daemon/api/architect.go` - spawnArchitectSession handler
- `internal/daemon/api/meta.go` - spawnMetaSession handler
- `internal/cli/tui/dashboard/model.go` - dashboard TUI spawn path

All changes have been rolled back. No source files were modified.