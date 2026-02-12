---
id: 55aea736-3401-4eab-90ab-b4cbd1eac07d
title: 'Debug: architect session still collides with cortex-meta after tmux fix'
type: debug
tags:
    - debug
    - tmux
    - session
references:
    - ticket:af35e04e-7e62-456f-8182-4326d77d6524
created: 2026-02-10T14:05:24.314668Z
updated: 2026-02-10T14:13:30.151196Z
---
## Problem

After the tmux trailing-colon fix (commit `43778b0`, ticket af35e04e), spawning an architect session for a project named "cortex" still incorrectly collides with an existing `cortex-meta` tmux session.

## Reproduction

1. Start a meta session (`cortex meta`) — creates tmux session `cortex-meta`
2. In the cortex project directory, run `cortex architect`
3. **Expected**: creates a new `cortex` tmux session with an `architect` window
4. **Actual**: reuses or updates the existing `cortex-meta` session

## What's already been ruled out

The tmux layer has been audited and fixed — all 8 session-targeted functions in `internal/tmux/` now append trailing colons for exact matching. Session store lookups (SessionManager) are per-project, keyed by filepath. Window name matching in `GetWindowByName` uses exact string comparison.

## What to investigate

The bug must be upstream of the tmux layer. Focus on:

1. **The full `Spawn()` handler in `internal/daemon/api/architect.go` (lines 86-252)** — trace what happens AFTER `DetectArchitectState` returns. How is the tmux session actually created? Does it call `CreateSession` or reuse an existing session?

2. **Spawn orchestration in `internal/core/spawn/`** — specifically `spawn.go` and `orchestrate.go`. When StateNormal is detected, what exact sequence of tmux calls creates the session and window? Is there a code path that bypasses the fixed functions?

3. **Session creation flow** — does the daemon create a new tmux session named "cortex", or does it create a window inside an existing session? If it tries to create a session that "already exists" (due to some other check), what fallback does it use?

4. **Meta session spawn flow in `internal/daemon/api/meta.go`** — compare how meta creates its `cortex-meta` session vs how architect creates its session. Are there shared code paths?

5. **Any tmux calls outside `internal/tmux/`** — are there direct `exec.Command("tmux", ...)` calls elsewhere that bypass the fixed Manager methods?

## Acceptance criteria

- Identify the exact code path causing the collision
- Fix the root cause
- Verify: with `cortex-meta` session running, `cortex architect` in the cortex project creates a separate `cortex` session
- All existing tests pass (`make test`, `make lint`)