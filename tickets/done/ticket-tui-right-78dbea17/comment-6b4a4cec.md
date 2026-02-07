---
id: 6b4a4cec-df97-4f88-8abd-af0bde7c15d8
author: claude
type: ticket_done
created: 2026-01-28T05:15:00.997333Z
---
## Summary

Implemented right panel navigation and detail modals for the ticket TUI. The right panel sidebar items (reviews and comments) are now a navigable cursor list with a centered detail modal overlay for viewing full content and taking actions.

## Files Modified

### `internal/cli/tui/ticket/keys.go`
- Added `KeyO` ("o") and `KeyEnter` ("enter") key constants
- Updated `helpText()` signature with `rightFocused bool` param — when right panel is focused, shows cursor-specific hints (`[j/k] select  [gg/G] first/last  [o/Enter] open`) instead of scroll hints
- Added `modalHelpText(isReview bool)` function for modal-specific help text

### `internal/cli/tui/ticket/styles.go`
- Added `sidebarSelectedStyle` with background color 237 for cursor highlight
- Added modal style block: `modalStyle` (rounded border, purple border color, padding), `modalHeaderStyle`, `modalSeparatorStyle`, `modalHelpStyle`, `modalRepoStyle`

### `internal/cli/tui/ticket/model.go`
- **Bug fix**: Changed `review.Summary` to `review.Title` in `renderSidebarReviews()` (broken by prior field rename in commit a4a0cfd)
- **New state fields**: `sidebarCursor`, `showDetailModal`, `modalViewport`, `modalIsReview`, `modalItemIndex`, `rejecting`
- **New message types**: `SessionRejectedMsg`, `RejectErrorMsg`
- **Helper methods**: `sidebarItemCount()`, `sidebarReviewCount()`, `isSidebarReview()`, `rejectSession()`
- **`handleRightPanelKey()`**: j/k cursor movement, gg/G first/last jump, o/Enter opens modal, plus global shortcuts (r refresh, x kill, s spawn, ga architect)
- **`handleDetailModalKey()`**: Esc/q close, j/k scroll viewport, a approve (review only), x reject (review only)
- **`openDetailModal()`**: Creates viewport sized ~60% width x ~70% height minus chrome, renders markdown content
- **`renderDetailModal()`**: Centered overlay using `lipgloss.Place` with header + separator + scrollable viewport + help bar
- **`renderModalHeader()`**: Review shows "Review Request · {time_ago}", Comment shows "{type} · {date}"
- **`renderModalContent()`**: Renders markdown with glamour; reviews include optional repo path line
- **Sidebar cursor highlight**: Both `renderSidebarReviews()` and `renderSidebarComments()` apply `sidebarSelectedStyle` at cursor position
- **Comments display**: Updated to use `comment.Title` with fallback to first line of content
- **Integration**: WindowSizeMsg resizes modal viewport, TicketLoadedMsg clamps cursor bounds, rejecting state in loading guard and View, detail modal check is highest priority in key dispatch

## Key Decisions

1. **Flat list ordering**: Reviews appear first in the cursor list, followed by comments, matching the sidebar rendering order
2. **Pointer receiver for openDetailModal**: Used `*Model` receiver since it mutates multiple fields; callers on value receivers can still call it since the local copy is addressable
3. **Rejection via AddComment**: No dedicated reject API exists; used `AddComment` with type "rejection" per existing codebase pattern
4. **Modal as full overlay**: Detail modal replaces the entire view content (returned early from View()) rather than rendering on top, keeping the implementation simple
5. **Markdown rendering per-modal**: Creates a fresh glamour renderer with modal-appropriate width rather than reusing the left panel renderer

## Verification

- `make build` — compiles cleanly
- `make lint` — 0 issues
- `make test` — all unit tests pass
- Merged to main via fast-forward