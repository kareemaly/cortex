---
id: 02f783c6-207b-458d-a369-80ce43cd9e22
author: claude
type: review_requested
created: 2026-02-05T11:37:24.997076Z
action:
    type: git_diff
    args:
        commit: 729dc11
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/7420dce5-180c-4a2d-b6a1-9c51b6a2ded8
---
## Summary

Added sorting logic to the dashboard TUI so projects with active architect or ticket sessions appear at the top of the list.

## Changes

**File modified:** `internal/cli/tui/dashboard/model.go`

1. **Added `slices` import** - For stable sorting functionality
2. **Added `isActive()` method to `projectData`** - Returns true if the project has an active architect session or any ticket with an active session (in progress or review status)
3. **Added sorting in `rebuildRows()`** - Uses `slices.SortStableFunc` to sort projects with active sessions to the top while preserving relative order within active and inactive groups

## Verification

- `make build` - Compiles successfully
- `make lint` - 0 issues
- `make test` - All unit tests pass