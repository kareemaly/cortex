---
id: 8371a62a-8d66-4429-a0e4-1c386e1b0e22
title: Highlight Entire Comment Row on Selection in TUI
type: work
created: 2026-02-02T16:32:01.20561Z
updated: 2026-02-02T16:51:23.855576Z
---
## Summary

When a comment is selected in the ticket TUI, only the comment type badge is highlighted. The entire comment row should be highlighted for better visibility.

## Current Behavior
- Comment type badge (e.g., "review_requested") is highlighted
- Rest of the comment row (timestamp, preview text) has normal styling

## Expected Behavior
- Entire comment row has background highlight when selected
- Makes it clear which comment is currently focused

## Location

`internal/cli/tui/ticket/model.go` — `renderCommentRow()` 

## Acceptance Criteria
- [ ] Selected comment has full-row background highlight
- [ ] Non-selected comments retain normal styling