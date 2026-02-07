---
id: 78745200-d211-448d-aee9-2ef5f5ff67a4
title: Fix tmux nested session handling
type: ""
created: 2026-01-21T08:38:43Z
updated: 2026-01-21T08:38:43Z
---
`cortex architect` fails when run from inside an existing tmux session with:
```
sessions should be nested with care, unset $TMUX to force
Error: failed to attach to session: exit status 1
```

## Context

The `AttachSession` function in `internal/tmux/session.go:62-77` uses `tmux attach-session`, which fails when already inside tmux.

## Requirements

1. **Detect tmux environment**
   - Check if `$TMUX` environment variable is set
   - If set, we're inside tmux

2. **Switch behavior based on context**
   - **Outside tmux**: Use `attach-session -t <session>` (current behavior)
   - **Inside tmux**: Use `switch-client -t <session>` to switch to the target session

3. **Also handle window focusing**
   - When switching to architect session, select window 0 (architect window)
   - Use `select-window -t <session>:0` before or after switch

## Files to modify

- `internal/tmux/session.go` - Update `AttachSession` to handle both cases

## Verification

```bash
make build
make lint
make test
```

## Implementation

### Commits pushed
- `e376ba1` fix: handle tmux nested session by using switch-client when inside tmux

### Key files changed
- `internal/tmux/session.go` - Added `IsInsideTmux()` helper and modified `AttachSession` to use `switch-client` when inside tmux

### Important decisions
- Window selection (`select-window -t <session>:0`) is non-fatal - if it fails, we still proceed with attach/switch
- Used `_, _ = m.run(...)` to explicitly ignore the error from window selection while satisfying the linter

### Scope changes
- None - implementation followed the original ticket requirements