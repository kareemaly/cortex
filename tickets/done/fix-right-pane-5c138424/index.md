---
id: 5c138424-ad4a-42cb-8d4d-1407b226734c
title: Fix Right Pane Working Directory in Worktree Mode
type: ""
created: 2026-01-26T17:33:45.141041Z
updated: 2026-01-26T18:13:21.808185Z
---
## Problem

When spawning a session in worktree mode, the right pane runs `cortex show` from the worktree directory (`~/.cortex/worktrees/{sessionID}/`) instead of the project directory. The `cortex show` command needs the project root to locate `.cortex/` and resolve the ticket.

Example:

```
➜  9ebfff61-4a80-4917-8821-22345d4ae311 CORTEX_TICKET_ID=63e9ca04-... cortex show
```

This runs inside `~/.cortex/worktrees/9ebfff61.../` but should run from `~/projects/cortex1`.

## Solution

When building the right pane command in the spawn flow, use the original project path as the working directory, not the worktree path. The worktree path should only apply to the left pane (the agent's working directory).

## Key Files

| File | Likely Change |
|------|--------------|
| `internal/core/spawn/spawn.go` | Set right pane working directory to project path |
| `internal/tmux/window.go` | May need to support per-pane working directories |

## Acceptance Criteria

- [ ] Right pane `cortex show` runs from the project directory in worktree mode
- [ ] Left pane (agent) still runs from the worktree directory