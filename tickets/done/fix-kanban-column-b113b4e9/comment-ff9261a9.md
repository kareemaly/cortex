---
id: ff9261a9-7f93-43c2-9f24-3524e1f2c34a
author: claude
type: done
created: 2026-02-06T09:46:22.169623Z
---
## Root Cause

lipgloss `Height()` on kanban columns (column.go:262-264) is a styling hint, not a content clipper. It pads short content to reach the target height but lets long content overflow past the boundary. This is a known lipgloss limitation (charmbracelet/lipgloss#528). When the Done column has many tickets with multi-line titles, rendered content exceeds the height constraint, leaks out, and pushes the entire terminal UI layout.

## Resolution

Replaced the manual index-based scroll system with `bubbles/viewport` per column in `internal/cli/tui/kanban/column.go`. The viewport properly clips content within a fixed height.

Changes:
- Replaced `scrollOffset` field with `vp viewport.Model`
- Added `renderAllTickets()` — renders all tickets into a single string for viewport content
- Added `cursorYOffset()` — calculates cursor Y-line position for scroll-to-cursor
- Rewrote `View()` to use viewport for clipped content display with scroll position preservation
- Simplified navigation methods (`JumpToFirst`, `JumpToLast`, `ScrollUp`, `ScrollDown`) to only adjust `cursor`
- Removed dead code: `scrollOffset`, `EnsureCursorVisible()`, `countVisibleTickets()`
- No changes to `model.go` — Column interface unchanged

Verified: `make build`, `make test`, `make lint` all pass. Merged to main (fast-forward).