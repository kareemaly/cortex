---
id: af35e04e-7e62-456f-8182-4326d77d6524
title: Fix tmux session prefix matching in SessionExists
type: debug
tags:
    - debug
    - tmux
    - session
created: 2026-02-10T07:11:06.455052Z
updated: 2026-02-10T09:42:02.923683Z
---
## Problem

When a `cortex-meta` tmux session exists and the user spawns an architect for a project named "cortex", the system incorrectly detects `cortex-meta` as an existing match. This is because `tmux has-session -t cortex` does **prefix matching** and matches `cortex-meta`.

## Root Cause

`SessionExists` in `internal/tmux/session.go` runs `has-session -t name` without a trailing colon. Tmux interprets this as a prefix match.

The codebase already accounts for this in `CreateWindow` (`internal/tmux/window.go`) which appends `":"` and has an explicit comment explaining the prefix-matching behavior. The same fix was never applied to `SessionExists`.

## Reproduction

1. Start a meta session (`cortex meta`) — creates `cortex-meta` tmux session
2. In a project named "cortex", run `cortex architect`
3. System incorrectly tries to reuse `cortex-meta` instead of creating a new session

## Acceptance Criteria

- `SessionExists` uses exact matching (trailing colon) consistent with `CreateWindow`
- Audit other tmux target references in `internal/tmux/` for the same issue (e.g. `ListWindows`, `KillSession`, etc.)
- Spawning an architect for project "cortex" works correctly when a `cortex-meta` session exists
- Existing tests pass; add a test case if feasible