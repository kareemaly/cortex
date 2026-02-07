---
id: bd84ba87-f063-4629-930e-31a0a92b96fe
title: Set Architect Agent Pane to 30% Width
type: ""
created: 2026-01-27T10:21:52.356345Z
updated: 2026-01-27T10:28:56.139573Z
---
## Problem

The architect agent pane width should be 30%, matching the ticket agent pane layout exactly.

## Context

Both `SpawnAgent` and `SpawnArchitect` in `internal/tmux/command.go` call `SplitWindowHorizontal` which uses `split-window -h -p 70`. This gives 70% to the new right pane and 30% to the left agent pane. The code paths appear identical, but the architect pane may not be rendering at the expected 30% width in practice.

Layout should be:
```
┌──────────┬────────────────────────┐
│ Agent    │  Companion (kanban)    │
│ 30%      │  70%                   │
└──────────┴────────────────────────┘
```

## Scope

- Investigate why the architect pane width differs from the ticket agent pane
- Ensure the architect agent pane is exactly 30% width, matching the ticket agent layout
- Relevant files:
  - `internal/tmux/pane.go` — `SplitWindowHorizontal` (split logic)
  - `internal/tmux/command.go` — `SpawnArchitect` and `SpawnAgent`
  - `internal/core/spawn/spawn.go` — `spawnInTmux`

## Acceptance Criteria

- [ ] Architect agent pane renders at 30% width
- [ ] Layout matches ticket agent pane exactly