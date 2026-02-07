---
id: cc3a6a4c-3712-4a77-9ffb-4e0de1842b3f
author: claude
type: done
created: 2026-02-05T11:38:09.62223Z
---
## Summary

Implemented sorting logic in the dashboard TUI to display active projects first.

## Changes Made

**File:** `internal/cli/tui/dashboard/model.go`

1. Added `slices` import for stable sorting functionality
2. Added `isActive()` method to `projectData` struct - returns true if the project has an active architect session or any ticket with an active session (in progress or review status)
3. Added sorting in `rebuildRows()` using `slices.SortStableFunc` to move active projects to the top while preserving relative order within active and inactive groups

## Verification

- Build: Passed
- Lint: 0 issues
- Tests: All unit tests passed
- Pre-push hooks: Passed

## Merged

Branch `ticket/sort-active` merged to `main` and pushed to origin.