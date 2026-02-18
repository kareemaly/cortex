---
id: 5fb23750-a606-4be0-bf32-21401f66396d
title: Truncate long titles in docs/config explorer pane
type: work
tags:
    - tui
    - docs
created: 2026-02-12T14:36:12.544935Z
updated: 2026-02-12T14:41:45.42394Z
---
## Problem

In the docs and config explorer panes (left side), long titles wrap to the next line without maintaining tree indentation. The wrapped text starts at column 0, breaking the tree-view structure and making the list unreadable.

Example of current broken behavior:
```
 ▼ sessions (8)
   ├─ Architect Session —
2026-02-12T13:05Z
   ├─ OpenCode System Prompt
Injection: Complete Guide
```

## Requirements

- Truncate item titles that exceed the available width of the explorer pane, appending an ellipsis (`…`)
- The truncation point should account for the indentation and tree characters (`├─`, `└─`) so the ellipsis lands at the pane edge
- Every list item should render as a single row — no wrapping
- This applies to both the docs explorer and the config explorer panes
- The full title is already visible in the preview pane when an item is selected, so no information is lost

Expected result:
```
 ▼ sessions (8)
   ├─ Architect Session — 2026-02…
   ├─ OpenCode System Prompt Inje…
```

## Acceptance Criteria

- Long titles are truncated with `…` instead of wrapping
- Truncation respects indentation level (deeper items have less horizontal space)
- Every item in the explorer is exactly one row tall
- Short titles that fit within the pane are unaffected
- Both docs and config explorer panes handle this consistently