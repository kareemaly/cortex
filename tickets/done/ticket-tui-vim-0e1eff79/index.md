---
id: 0e1eff79-739d-44b4-8091-9c7a78595bb6
title: Ticket TUI Vim Scrolling
type: ""
created: 2026-01-24T09:39:07Z
updated: 2026-01-24T09:39:07Z
---
## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

Ticket detail view only has basic j/k scrolling. Missing vim-style page navigation.

## Requirements

Add vim shortcuts to ticket detail TUI:
- `ctrl+u` / `ctrl+d` - scroll half page up/down
- `gg` - jump to top
- `G` - jump to bottom

## Implementation

### Commits
- `2f2cfbf` feat: add vim-style scrolling to ticket detail TUI

### Key Files Changed
- `internal/cli/tui/ticket/keys.go` - Added key constants (KeyCtrlU, KeyCtrlD, KeyG, KeyShiftG) and updated help text
- `internal/cli/tui/ticket/model.go` - Added pendingG state field and key handlers for all new shortcuts

### Decisions
- Used `HalfPageUp()`/`HalfPageDown()` from viewport (non-deprecated methods) for half-page scroll
- Followed existing kanban implementation pattern for `gg` sequence with `pendingG` state tracking
- Updated help bar to show `[j/k/gg/G] scroll  [ctrl+u/d] page` for discoverability