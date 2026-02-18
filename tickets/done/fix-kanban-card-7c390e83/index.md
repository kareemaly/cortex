---
id: 7c390e83-34c5-4897-8042-5e8d1accc354
title: Fix kanban card highlight not covering type badge
type: debug
tags:
    - tui
created: 2026-02-12T15:00:12.564635Z
updated: 2026-02-13T08:38:25.218578Z
---
## Bug

The selection highlight on kanban ticket cards doesn't cover the `[work]` (or other type) badge. The purple/blue background starts only from the title text, leaving the type badge unstyled on the first line. This creates a visual break in the highlight.

This started happening after the ticket type badge (e.g. `[work]`, `[debug]`, `[research]`) was added next to the title.

## Reproduction

1. Open kanban TUI (`cortex kanban`)
2. Navigate to any ticket card that has a type badge
3. Observe the highlight background does not extend behind the `[work]` badge — only the title portion is highlighted

## Expected Behavior

The selection highlight background should cover the entire card uniformly, including the type badge. The badge text color (green, etc.) should render on top of the highlight background.

## Likely Cause

The type badge has its own lipgloss style that sets foreground color but doesn't inherit/preserve the selection background. When the card is selected, the badge style likely overrides or ignores the background color applied to the rest of the card.

## Acceptance Criteria

- Selection highlight background covers the full card including the type badge
- Type badge retains its colored text but renders on top of the highlight background
- Works for all ticket types (work, debug, research, chore)