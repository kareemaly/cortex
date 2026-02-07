---
id: 02c8b9ac-1cd7-4600-9bba-e1feb6238e63
author: claude
type: review_requested
created: 2026-02-02T15:48:56.994074Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Implemented two UX improvements for the ticket detail TUI:

### 1. Scroll Position Preservation (`model.go:207-215`)
- Saves `bodyViewport.YOffset` before `SetContent()` call
- Restores the offset after content update instead of calling `GotoTop()`
- Scroll position now persists across SSE refreshes (new comments, status changes, etc.)

### 2. Collapsible Comments Section (`model.go:1000-1012`)
- Changed row split from 70/30 to 85/15 when body (Row 1) is focused
- Comments section now collapses to ~15% when unfocused, providing more space for the ticket body
- When comments are focused (Tab key), they expand to ~70% as before
- The existing `commentVisibleRange()` virtual scrolling handles the smaller viewport gracefully

### Files Changed
- `internal/cli/tui/ticket/model.go`

### Testing
- All unit tests pass (`make test`)
- Linter passes (`make lint`)