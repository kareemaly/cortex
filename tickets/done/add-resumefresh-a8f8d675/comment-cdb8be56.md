---
id: cdb8be56-72a0-4fea-a405-c75a649d5d8c
author: claude
type: review_requested
created: 2026-02-07T12:54:28.55653Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
        commit: 555408a
---
## Summary

Added a mode selection modal for orphaned architect sessions in the dashboard TUI. Previously, pressing `[s]` on an orphaned architect would call `spawnArchitect("")` with no mode, resulting in a silent 409 error. Now it shows an inline modal prompting the user to choose.

## Changes (single file: `internal/cli/tui/dashboard/model.go`)

1. **New model fields**: `showArchitectModeModal` and `architectModeProjectPath` to track modal state
2. **New `handleArchitectModeKey` function**: Handles `[r]` for resume, `[f]` for fresh, `[esc]` to cancel
3. **New `spawnArchitectWithMode` function**: Like `spawnArchitect` but passes an explicit mode string
4. **Modified `handleKeyMsg`**: Intercepts all keys when modal is active (same pattern as unlink confirmation)
5. **Modified `handleSpawnArchitect`**: Detects orphaned state and shows modal instead of spawning directly
6. **Modified `handleFocusCurrentRow`**: Orphaned branch now triggers modal instead of showing a passive status message
7. **Modified `View`**: Renders modal dialog with warn badge style and help bar showing options

## Verification

- `make build` — compiles cleanly
- `make lint` — 0 issues
- `make test` — all unit tests pass