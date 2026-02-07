---
id: d0343b8c-348a-4d2a-921f-79f8d5d9aada
author: claude
type: review_requested
created: 2026-02-05T11:37:17.427326Z
action:
    type: git_diff
    args:
        commit: 287dd08
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/61b8b2b3-4377-4235-a147-08bfc1669981
---
## Summary
Added the "work" ticket type badge to the kanban TUI for consistency with other ticket types.

## Changes
1. **styles.go**: Added `workTypeBadgeStyle` with cyan/teal color (35) to distinguish work tickets
2. **styles.go**: Added "work" case to `typeBadgeStyle()` function
3. **column.go**: Removed the `&& t.Type != "work"` exclusion condition so work badges are now displayed

## Testing
- `make build` - passes
- `make test` - all unit tests pass

## Visual Result
Work tickets will now show a cyan `[work]` badge prefix, similar to:
- `[debug]` (red)
- `[research]` (blue)
- `[chore]` (gray)