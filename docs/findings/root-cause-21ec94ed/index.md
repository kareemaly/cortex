---
id: 21ec94ed-9188-4b86-ba0f-3a07345a201d
title: 'Root Cause: Architect Tmux Pane Split 50/50 Instead of 30/70'
tags:
    - tmux
    - architect
    - pane-split
    - root-cause
created: 2026-02-13T09:35:58.771163Z
updated: 2026-02-13T09:35:58.771163Z
---
## Summary

The architect session tmux pane split appears as ~42-50% / 50-58% instead of the intended 30% / 70%. The root cause is **tmux's even (non-proportional) space distribution during window resize** when a client attaches to a detached session.

## Root Cause

### The Problem Flow (Architect)

1. `cortex architect` → CLI calls `POST /architect/spawn` to daemon
2. Daemon's `SpawnArchitect()` calls `tmux new-session -d -s <name>` → **detached** session, default size **80x24**
3. `split-window -h -p 70` creates correct 30/70 split at 80 columns (23 cols / 56 cols)
4. Daemon returns HTTP response
5. CLI calls `tmux attach-session` → window resizes to match terminal (e.g., 200 columns)
6. **Tmux distributes the extra 120 columns evenly (~60 to each pane)**, not proportionally
7. Result: 83 cols / 116 cols → **41.7% / 58.3%** (broken)

### Why Ticket Agents Are Unaffected

The ticket agent flow differs in a critical way:

1. The tmux session already exists (architect is running, client is attached)
2. `SpawnAgent()` calls `tmux new-window` → new window inherits attached client's terminal size (e.g., 200 cols)
3. `split-window -h -p 70` creates correct 30/70 split at the actual terminal width
4. No resize occurs → **29.6% / 70.4%** (correct)

### Tmux Resize Behavior

Tmux's layout resize algorithm distributes extra space **evenly** among panes, not proportionally. This is a fundamental tmux behavior (confirmed on tmux 3.6a):

| Initial Width | Terminal Width | Pane 0 | Pane 1 | Notes |
|:---:|:---:|:---:|:---:|:---|
| 80 | 120 | 36.1% | 63.9% | Drifting from 30/70 |
| 80 | 160 | 39.6% | 60.4% | |
| 80 | 200 | 41.7% | 58.3% | Typical terminal |
| 80 | 240 | 43.1% | 56.9% | |
| 80 | 300 | 44.5% | 55.5% | Approaches 50/50 |

**Control test** — creating session at the correct width produces perfect ratios:

| Created Width | Pane 0 | Pane 1 |
|:---:|:---:|:---:|
| 120 | 29.4% | 70.6% |
| 160 | 29.6% | 70.4% |
| 200 | 29.6% | 70.4% |
| 240 | 29.7% | 70.3% |
| 300 | 29.8% | 70.2% |

## Code Path Analysis

### Architect Session Creation

```
CLI: cmd/cortex/commands/architect.go:61
  → SDK: client.SpawnArchitect(mode)
    → HTTP POST /architect/spawn
      → Daemon: internal/daemon/api/architect.go:186 (spawnArchitectSession)
        → Spawner: internal/core/spawn/spawn.go:147 (Spawn)
          → internal/core/spawn/spawn.go:961 (spawnInTmux)
            → Tmux: internal/tmux/command.go:95 (SpawnArchitect)
              → internal/tmux/session.go:32 (CreateSession) ← detached, no size
              → internal/tmux/window.go:68 (CreateArchitectWindow)
              → internal/tmux/pane.go:40 (ResetWindowPanes)
              → internal/tmux/pane.go:10 (SplitWindowHorizontal) ← -p 70
CLI: cmd/cortex/commands/architect.go:87
  → manager.AttachSession(resp.TmuxSession) ← triggers resize, breaks ratio
```

### Ticket Agent Session Creation

```
Spawner: internal/core/spawn/spawn.go:955 (spawnInTmux)
  → Tmux: internal/tmux/command.go:53 (SpawnAgent)
    → internal/tmux/session.go:32 (CreateSession) ← no-op, session exists
    → internal/tmux/window.go:31 (CreateWindow) ← inherits attached client size
    → internal/tmux/pane.go:10 (SplitWindowHorizontal) ← -p 70 at correct size
```

### Key Difference

| | Architect | Ticket Agent |
|---|---|---|
| Session creation | New detached session (80x24 default) | Session already exists with attached client |
| Window creation | Renames existing window 0 | Creates new window (inherits terminal size) |
| Split happens at | 80 columns (default) | Actual terminal width |
| Client attach | Triggers resize → breaks ratio | N/A (already attached) |

### Why Previous Investigations Missed This

- Ticket `bd84ba87`: Found stale panes issue and added `ResetWindowPanes` — a real fix but for a different problem
- Ticket `680d88da`: Concluded "code is correct, 50/50 is from old sessions" — integration tests verified the ratio on the detached session but never tested after client attachment
- Integration test `TestIntegrationSpawnArchitectWithCompanion` (line 501): Verifies pane widths on the detached session, not after resize/attachment

### Critical File: `internal/tmux/session.go:41`

```go
args := []string{"new-session", "-d", "-s", name}
// ← No -x or -y flags — uses default 80x24
```

## Recommended Fixes

### Option A: Pass Terminal Size at Session Creation (Recommended)

Have the CLI detect the terminal size and include it in the spawn request. The daemon passes it through to `CreateSession`, which uses `-x width -y height`.

**Pros**: Root cause fix — split happens at correct size from the start  
**Cons**: Requires API change, CLI must detect terminal size

Changes needed:
1. `cmd/cortex/commands/architect.go` — detect terminal size, send in request
2. `internal/cli/sdk/client.go` — add size params to `SpawnArchitect()`
3. `internal/daemon/api/architect.go` — extract size from request, pass to spawner
4. `internal/core/spawn/spawn.go` — pass size to `SpawnArchitect()`
5. `internal/tmux/command.go` — `SpawnArchitect()` accepts optional size
6. `internal/tmux/session.go` — `CreateSession()` accepts optional `-x`/`-y`

### Option B: Resize Panes After Attach via Tmux Hook

Set a tmux `after-client-attached` hook in `SpawnArchitect` that resizes panes to the correct ratio when a client attaches.

```go
// After split in SpawnArchitect:
hookTarget := fmt.Sprintf("%s:%d.1", sessionTarget(session), ArchitectWindowIndex)
m.run("set-hook", "-t", sessionTarget(session)+":", "after-client-attached[99]",
    fmt.Sprintf("resize-pane -t %s -x 70%%", hookTarget))
```

**Pros**: Minimal code changes, works retroactively on every attach  
**Cons**: Relies on tmux hooks (version-dependent), adds external state

### Option C: CLI Resizes After Spawn (Before Attach)

In the CLI, before calling `AttachSession`, set the correct pane sizes using `resize-pane -x 70%`. But this doesn't help because the window is still at default size at this point.

**Note**: This doesn't work because the resize happens at default 80 columns, not the terminal size. Dismissed.

### Verification

After any fix, verify with:
```bash
# Create session, split, resize, check ratio
tmux new-session -d -s test -x $(tput cols) -y $(tput lines)
tmux split-window -h -p 70 -t test
tmux list-panes -t test -F '#{pane_index}:#{pane_width}'
# Should show ~30/70 ratio
tmux kill-session -t test
```

Integration tests should be updated to test the resize scenario:
1. Create detached session at small size
2. Split at 70%
3. Resize window to large size
4. Verify ratio is still ~30/70