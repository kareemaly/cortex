---
id: f37af025-250d-4243-aea2-2ca9ec5b79ee
author: claude
type: ticket_done
created: 2026-01-27T15:44:22.789999Z
---
## Summary

Implemented a 70/30 horizontal split panel layout for the ticket detail TUI (`cortex show`). The left panel shows the scrollable glamour-rendered ticket description, while the right panel displays a static metadata sidebar with details, session info, reviews, and comments. Panel focus is toggleable with `h`/`l` keys. The layout gracefully falls back to the existing single-column stacked layout when the terminal is narrower than 100 columns.

## Files Modified

- **`internal/cli/tui/ticket/styles.go`** (+46 lines) — Added `focusColor` (purple, matching header), split layout styles: `leftPanelFocusedStyle`/`leftPanelStyle`, `sidebarFocusedStyle`/`sidebarStyle`, `sidebarHeaderStyle`, `sidebarLabelStyle`, `sidebarValueStyle`, `sidebarDotStyle`, `dividerStyle`.

- **`internal/cli/tui/ticket/keys.go`** (+21/-9 lines) — Added `KeyH`/`KeyL` constants. Updated `helpText` signature to accept `splitLayout bool`, prepending `[h/l] panel` hint in split mode. Separated `[ga] architect` from the scroll group for clarity.

- **`internal/cli/tui/ticket/model.go`** (+333/-21 lines) — Renamed `viewport` → `leftViewport`. Added `focusedPanel`, `splitLayout` fields and `minSplitWidth` constant. Added `leftPanelWidth()`/`rightPanelWidth()` helpers. Updated `WindowSizeMsg` handler to set split mode and size viewport/renderer to left panel width. Added `renderLeftContent()`, `renderSplitLayout()`, `renderPanelDivider()`, `renderSidebar()`, `renderSidebarDetails()`, `renderSidebarSession()`, `renderSidebarReviews()`, `renderSidebarComments()`. Added `h`/`l` key handling for panel focus switching.

## Key Decisions

- **70/30 split ratio**: Left panel gets 70% of terminal width, right gets remainder minus 1 column for the divider. This gives the markdown body ample room while keeping the sidebar compact.
- **Focus indicator**: Purple left border on the focused panel, matching the header color. Unfocused panel uses padding instead of a border to maintain alignment.
- **Sidebar content**: One-liner format for reviews (repo · summary) and comments (type · content) with truncation to fit panel width. Review repo paths use `filepath.Base()` to save space.
- **Narrow fallback at 100 columns**: Below this threshold, the layout reverts to the existing single-column stacked view. The `h`/`l` keys are not consumed in narrow mode.
- **Scroll always targets left viewport**: Regardless of panel focus, scroll keys operate on the left (description) viewport. Right-panel scrolling is deferred to a follow-up ticket.

## Verification

- `make build` — compiles cleanly
- `make lint` — 0 issues
- `make test` — all unit tests pass

## Follow-up Notes

- Right-panel scrolling (when focused) could be added in a follow-up ticket for navigating long review/comment lists.
- The sidebar currently renders all sections statically; a future enhancement could add collapsible sections.