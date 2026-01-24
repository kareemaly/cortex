# Kanban Column Scrolling

## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

When a column (especially Done) has many tickets, the entire kanban UI overflows the terminal. Columns should have a max height with internal scrolling.

## Current State

- `column.go:77` - `View()` receives `maxHeight` and applies `.Height(maxHeight)` to the style
- But it renders ALL tickets without any viewport/windowing
- Only j/k navigation exists (single ticket up/down)

## Requirements

### Column Viewport

- Render only visible tickets that fit within `maxHeight`
- Track scroll offset in Column struct
- Auto-scroll to keep cursor visible when navigating

### New Vim Keybindings

Add to `keys.go` and handle in `model.go`:

| Key | Action |
|-----|--------|
| `ctrl+u` | Scroll up 10 tickets |
| `ctrl+d` | Scroll down 10 tickets |
| `gg` | Jump to first ticket |
| `G` | Jump to last ticket |

Note: `gg` requires tracking key sequences (first `g` sets pending state, second `g` triggers action).

### Visual Feedback

Show scroll indicators when content overflows:
- `▲` at top when scrolled down
- `▼` at bottom when more content below

## Files to Modify

- `internal/cli/tui/kanban/column.go` - Add scroll offset, viewport rendering
- `internal/cli/tui/kanban/keys.go` - Add new key constants
- `internal/cli/tui/kanban/model.go` - Handle new keys, track `g` pending state

## Implementation Notes

Calculate visible ticket count: `(maxHeight - headerLines - marginLines)`

Scroll offset adjusts when:
- Cursor moves above visible area: `offset = cursor`
- Cursor moves below visible area: `offset = cursor - visibleCount + 1`
- ctrl+u/ctrl+d: move cursor AND offset together
