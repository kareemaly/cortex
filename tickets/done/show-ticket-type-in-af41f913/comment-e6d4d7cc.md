---
id: e6d4d7cc-ea6d-492e-bf44-b713479aa6dd
author: claude
type: review_requested
created: 2026-02-05T11:37:53.210884Z
action:
    type: git_diff
    args:
        commit: 4f75653
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/43eb5b71-9a3c-4f13-b313-2601be65ac28
---
## Summary

Added ticket type display to the `cortex show` command in both TUI and plain-text output modes, following the existing kanban type badge pattern.

## Changes

### 1. `internal/cli/tui/ticket/styles.go`
- Added type badge styles matching kanban: `debugTypeBadgeStyle` (red), `researchTypeBadgeStyle` (blue), `choreTypeBadgeStyle` (gray)
- Added `typeBadgeStyle()` function to return the appropriate style for a ticket type

### 2. `internal/cli/tui/ticket/model.go`
- Updated `renderHeader()` to include type badge between ID and title
- Type badge only shown for non-"work" types (consistent with kanban behavior)
- Format: `<id> [<type>] <title> ... <status>`

### 3. `cmd/cortex/commands/ticket_show.go`
- Added Type line after Status in `printTicketPlainText()` for non-TTY output

## Verification
- Build passes: `make build`
- Lint passes: `make lint` (0 issues)
- Tests pass: `make test`