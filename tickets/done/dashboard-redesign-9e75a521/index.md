---
id: 9e75a521-08ed-44de-aeae-61b7ff146501
title: 'Dashboard Redesign: Flatten Layout, Remove Collapse, Fix Column Widths'
type: ""
created: 2026-01-27T13:00:29.088806Z
updated: 2026-01-27T13:14:58.464692Z
---
## Summary

Redesign the cortex dashboard TUI to remove the expand/collapse tree behavior and simplify the layout. All projects and their sessions should always be visible.

## Changes

### 1. Remove collapse/expand behavior
- All projects show their sessions immediately — no expand/collapse toggle
- Remove `expanded` state from `projectData`
- Eagerly load all project details on init
- Keep SSE connections open for all projects (no per-project subscribe/unsubscribe on toggle)

### 2. Architect becomes a project-level indicator, not a session row
- Remove the architect session row from the session list entirely
- Replace the `▶`/`▼` expand indicator on the project row with `●`/`○` to indicate architect active/inactive
- Projects without an active architect should appear visually dimmed/inactive
- Pressing `Enter`/`F` on a project row focuses the architect (if active); `S` spawns one

### 3. Fix column widths to be constant
- Session rows currently shift width when selected because `selectedStyle.Width(m.width - 1)` stretches the full row
- Apply the same fixed column formatting (title, status, duration) to both selected and unselected rows
- Selection should only change the background highlight, not the layout

### 4. Remove broken architect duration
- Architect `StartedAt` is unreliable (shows `106751d 23h` due to zero-value time)
- Since architect is no longer a session row, this is resolved by removal

## Files
- `internal/cli/tui/dashboard/model.go` — model, update, view, rebuildRows
- `internal/cli/tui/dashboard/styles.go` — add inactive project style
- `internal/cli/tui/dashboard/keys.go` — simplify keybindings (remove H for collapse)