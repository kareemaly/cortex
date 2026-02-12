---
id: 78644b1a-330e-4686-9ab9-d5be1d56c009
title: 'Debug: architect spawn reports "already spawned" due to cortex-meta session'
type: debug
tags:
    - debug
    - tmux
    - session
    - architect
references:
    - ticket:55aea736-3401-4eab-90ab-b4cbd1eac07d
    - ticket:af35e04e-7e62-456f-8182-4326d77d6524
created: 2026-02-10T14:13:43.858126Z
updated: 2026-02-10T14:52:24.867274Z
---
## Problem

When spawning an architect session for a project named "cortex", the system reports "already spawned" even though no architect session exists. The only running tmux session is `cortex-meta` (the meta agent session).

## Reproduction

1. Start a meta session (`cortex meta`) — creates tmux session `cortex-meta`
2. Run `cortex architect` in the cortex project directory
3. **Expected**: creates a new `cortex` tmux session with an `architect` window
4. **Actual**: reports "already spawned" and does not create the architect session

## Root cause hypothesis

The spawn state detection logic (`DetectArchitectState` or upstream) likely checks for an existing tmux session using a prefix match or `has-session` without exact matching. When checking for session `cortex`, tmux matches `cortex-meta` as a prefix hit, causing the system to believe an architect session already exists.

## What's already been ruled out

- The 8 session-targeted tmux Manager functions in `internal/tmux/` were already fixed to use trailing colons for exact matching (commit `43778b0`, ticket af35e04e)
- The bug is upstream of those functions — likely in how `DetectArchitectState` or the spawn handler checks session existence

## What to investigate

1. **`DetectArchitectState`** in `internal/core/spawn/` — how does it determine if an architect session is already active? Does it call a tmux function that was missed in the trailing-colon fix?
2. **Session existence checks** — are there any `tmux has-session` or `tmux list-sessions` calls that don't use exact matching (trailing colon)?
3. **SessionManager lookups** — does the in-memory session store incorrectly match "cortex" to the meta session?
4. **Any tmux calls outside `internal/tmux/`** — direct `exec.Command("tmux", ...)` calls that bypass the fixed Manager methods

## Acceptance criteria

- Identify why spawn reports "already spawned" when only `cortex-meta` exists
- Fix the root cause so `cortex architect` correctly creates a separate session
- Verify: with `cortex-meta` running, `cortex architect` spawns successfully
- All tests pass (`make test`, `make lint`)