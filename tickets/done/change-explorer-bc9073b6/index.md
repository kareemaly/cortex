---
id: bc9073b6-bc37-41d7-8f5a-e233d799e597
title: Change explorer selection highlight to accent-colored text
type: work
tags:
    - tui
    - docs
    - configuration
created: 2026-02-12T14:58:05.761685Z
updated: 2026-02-13T08:31:52.220742Z
---
## Problem

The current selection indicator in the docs and config explorer panes uses a thin `▎` vertical bar character on the left side of the selected item. This is subtle and hard to see.

## Requirements

- Replace the current selection indicator with accent-colored text on the selected item's filename/title
- Remove the `▎` bar indicator entirely — use a regular space for all items (selected or not)
- When an item is selected and the explorer is focused: render the title text in the accent color (currently color 62, purple/blue) + bold
- When an item is selected but the explorer is NOT focused: render in a muted/dimmer style (e.g. color 245 gray text, no bar)
- When not selected: default styling, no change
- Apply consistently to both the docs explorer and the config explorer panes

Both panes use identical selection logic today — they should stay consistent after this change.

## Acceptance Criteria

- Selected item in focused explorer shows accent-colored bold text (no bar character)
- Selected item in unfocused explorer shows muted text (no bar character)
- The `▎` indicator is no longer rendered anywhere in the explorer panes
- Both docs and config explorers behave identically
- No visual regression on unselected items