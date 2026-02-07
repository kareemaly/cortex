---
id: 68e13c68-4fcd-469d-bf5f-ec99ac464d80
title: Kanban Card Spacing and Layout Fixes
type: ""
created: 2026-01-27T10:05:33.707926Z
updated: 2026-01-27T10:17:09.978222Z
---
## Problem

Several cosmetic issues with the kanban ticket card layout:

1. **Unnecessary left/right padding on ticket text** — After removing the icon prefix, there are still 2 characters of left padding (`  `) on every title line and the metadata line. This wastes horizontal space in already narrow columns.

2. **No vertical space between tickets** — Ticket cards are stacked directly on top of each other with no gap, making it hard to distinguish where one ticket ends and the next begins.

3. **Extra top margin on first ticket in Done column** — The first ticket in the Done column has extra vertical space above it that other columns don't have.

4. **Premature scroll indicator** — The scroll bar/indicator appears before tickets have filled the full vertical space of the column, wasting available space.

## Scope

- **`internal/cli/tui/kanban/column.go`**:
  - Remove the 2-char left indent from title lines and metadata line
  - Add vertical spacing (empty line) between ticket cards
  - Fix top margin inconsistency in the Done column's first ticket
  - Fix visible ticket count calculation so tickets fill the full column height before scroll indicators appear

## Acceptance Criteria

- [ ] Ticket title and metadata text use full available width (no extra left/right padding)
- [ ] Visible vertical gap between ticket cards
- [ ] No extra top margin on the first ticket in any column
- [ ] Scroll indicators only appear when tickets actually overflow the column height