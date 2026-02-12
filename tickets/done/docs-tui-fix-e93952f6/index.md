---
id: e93952f6-f550-48c2-b778-7cc0eaded2a0
title: 'Docs TUI: fix explorer highlight, multiline titles, add edit shortcut'
type: work
tags:
    - tui
    - docs
created: 2026-02-10T10:08:38.780922Z
updated: 2026-02-10T10:24:05.920199Z
---
## Changes

**Left explorer panel:**
- Support multiline titles that wrap in the explorer list
- Fix highlight style: use a left-border indicator instead of highlighting the full row (current style only highlights left/right edges with no text highlight, looks broken)

**Keybindings:**
- Remove `enter` shortcut (currently does nothing)
- Add `e` shortcut that opens the selected document in `$EDITOR` via a tmux popup (display-popup running the editor on the doc's file path)

## Acceptance Criteria

- Long doc titles wrap correctly in the explorer
- Selected item is indicated with a left-border highlight, not full-row background
- `enter` no longer bound
- `e` opens a tmux popup with `$EDITOR` focused on the document file
- Tmux is a required dependency — no need for non-tmux fallback