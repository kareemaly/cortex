---
id: 8fd3b399-53c5-4760-9cba-d28010b425eb
author: claude
type: review_requested
created: 2026-02-06T09:46:02.066907Z
action:
    type: git_diff
    args:
        commit: "6615715"
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/44384640-ea6b-4810-a3bc-7e7277dde1d5
---
## Root Cause

lipgloss `Height()` (used on column.go:262-264) is a styling hint, not a content clipper. It pads short content to reach the target height but lets long content overflow past the boundary. This is a known lipgloss limitation (charmbracelet/lipgloss#528). When the Done column has many tickets with multi-line titles, content overflows and pushes the entire terminal UI layout.

## Fix

Replaced the manual index-based scroll system with `bubbles/viewport` per column. The viewport properly clips content within a fixed height.

### Changes in `internal/cli/tui/kanban/column.go`:

1. **Replaced `scrollOffset` field with `vp viewport.Model`** in the Column struct
2. **Added `renderAllTickets()`** — renders ALL tickets into a single string (same rendering logic, just iterates all tickets instead of a visible window)
3. **Added `cursorYOffset()`** — calculates the Y-line where the cursor's ticket starts, used for scroll-to-cursor logic
4. **Rewrote `View()`** — sets viewport dimensions, renders content via `renderAllTickets()`, preserves scroll position across re-renders (`SaveYOffset`/`SetYOffset`), ensures cursor visibility by adjusting `vp.YOffset`, outputs header + scroll indicators + `vp.View()`. `Height(maxHeight)` is kept for uniform column padding (safe now since viewport already clips)
5. **Simplified navigation** — `JumpToFirst`, `JumpToLast`, `ScrollUp`, `ScrollDown` now only adjust `cursor`; viewport scroll is handled in `View()`
6. **Removed dead code** — `scrollOffset` field, `EnsureCursorVisible()`, `countVisibleTickets()`, scrollOffset bounds check in `SetTickets()`

### No changes to model.go — the Column interface stays identical. No viewport message routing needed since we drive the viewport imperatively in View().

## Verification

- `make build` — compiles cleanly
- `make test` — all tests pass
- `make lint` — 0 issues