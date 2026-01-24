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

## Implementation

### Commits Pushed

- `d4651c7` feat: implement viewport scrolling with vim navigation for kanban columns
- `928e083` Merge branch 'ticket/2026-01-23-kanban-column-scrolling'
- `ad80c58` fix: only reserve scroll indicator space when scrolling is needed

### Key Files Changed

- `internal/cli/tui/kanban/column.go` - Added `scrollOffset` field, viewport rendering with scroll indicators, navigation methods (`JumpToFirst`, `JumpToLast`, `ScrollUp`, `ScrollDown`, `EnsureCursorVisible`)
- `internal/cli/tui/kanban/keys.go` - Added key constants (`KeyCtrlU`, `KeyCtrlD`, `KeyG`, `KeyShiftG`), updated help text
- `internal/cli/tui/kanban/model.go` - Added `pendingG` state for gg sequence, key handlers for new navigation
- `internal/cli/tui/kanban/styles.go` - Added `mutedStyle` for scroll indicators

### Important Decisions

- Scroll indicators (▲/▼) only reserve space when scrolling is actually needed (fixed in ad80c58)
- `visibleCount` calculated as `maxHeight - headerLines - indicatorLines` (indicatorLines=0 when no scrolling needed)
- Scroll amount for ctrl+u/d set to 10 tickets (half-page feel)
- `EnsureCursorVisible` called during rendering to ensure cursor stays in viewport

### Scope Changes

None - implemented as specified in ticket requirements
