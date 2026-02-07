---
id: 0ba03aa4-61c0-48a8-9cbe-16de9473b9d9
title: 'Fix kanban viewport scroll: highlight and chevron issues'
type: debug
created: 2026-02-06T10:11:56.661433Z
updated: 2026-02-06T10:21:45.312702Z
---
## Problem

Two issues after the viewport migration (b113b4e9):

1. **Highlight disappears on scroll**: When scrolling beyond the first page of tickets in a column, the focused/highlighted ticket style disappears. The viewport scroll position and cursor position are likely out of sync — the cursor moves but the viewport isn't scrolling to keep the highlighted ticket visible, or the highlight offset calculation doesn't account for the viewport scroll offset.

2. **Bottom chevron stays visible at end**: When jumping to the last ticket with "G", the bottom scroll indicator chevron (▼) is still shown even though there's no more content below. The chevron visibility check needs to account for the viewport being scrolled to the bottom.

## Files

- `internal/cli/tui/kanban/column.go` — cursor tracking, highlight rendering, chevron logic, viewport scroll sync
- `internal/cli/tui/kanban/model.go` — key handling for "G" and scroll commands

## Context

These are regressions from the viewport migration in ticket b113b4e9. The previous manual scroll logic handled these cases; the new viewport-based approach needs equivalent behavior.