---
id: 75c8d76f-e363-43ed-86d3-8a0ffcb5cb6a
author: claude
type: ticket_done
created: 2026-01-27T13:14:58.460806Z
---
## Dashboard Redesign: Flatten Layout, Remove Collapse, Fix Column Widths

### Summary
Redesigned the cortex dashboard TUI to remove the expand/collapse tree behavior, making all projects and their active sessions immediately visible. The architect session is now represented as a project-level indicator rather than a separate row, and column widths no longer shift on selection.

### Files Modified
- `internal/cli/tui/dashboard/styles.go` — Added `dimmedProjectStyle`, removed `runningBadgeStyle` and `greenColor`
- `internal/cli/tui/dashboard/keys.go` — Removed `KeyH` constant, updated help text
- `internal/cli/tui/dashboard/model.go` — Core layout and behavior changes (net -107 lines)

### Key Changes

**1. Removed expand/collapse behavior**
- Removed `expanded` field from `projectData` struct
- `ProjectsLoadedMsg` handler now eagerly dispatches `loadProjectDetail()` + `subscribeProjectEvents()` for every existing project
- SSE events and spawn messages always trigger reloads (removed `expanded` guard)
- Deleted `handleExpandCollapse()` and `handleCollapse()` functions
- Removed `KeyH` (collapse key) from keybindings

**2. Architect as project-level indicator**
- Removed `isArchitect` field from `row` struct
- `rebuildRows()` no longer emits architect session rows
- Project rows now show `●` (pink, active architect) or `○` (gray, inactive) instead of `▶`/`▼`
- Projects without an active architect render with dimmed styling (`dimmedProjectStyle`)
- Removed `runningBadgeStyle` and `greenColor` (only used by architect session row)

**3. Focus replaces expand**
- New `handleFocusCurrentRow()` replaces the three old handlers
- Enter/L/F on project row: focuses architect if active, shows "No active architect. Press [s] to spawn." otherwise, shows "Project is stale" for stale projects
- Enter/L/F on session row: focuses the ticket session
- Help text updated to `[enter/f] focus  [s]pawn architect  [r]efresh  [j/k/gg/G] navigate  [q]uit`

**4. Fixed column width shifts**
- Removed `selectedStyle.Width(m.width - 1)` from all render paths (project rows and session rows)
- Selection now only changes background color, not layout dimensions

### Verification
- `make build` — passes
- `make lint` — passes (0 issues)
- `make test` — all unit tests pass

### Decisions
- Eager loading all projects on init rather than lazy loading: simplifies the model and provides immediate visibility into all project states
- Using `●`/`○` for architect status: provides clear visual distinction without taking up a full row
- Dimming inactive projects: helps users quickly identify which projects have active architect sessions

### Commit
`05a47d7` — feat: flatten dashboard layout, remove collapse, fix column widths (merged to main via fast-forward)