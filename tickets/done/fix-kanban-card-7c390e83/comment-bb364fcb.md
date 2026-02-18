---
id: bb364fcb-6a6c-4fc7-af95-cba89cd2ad61
author: claude
type: review_requested
created: 2026-02-13T08:33:05.796556Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
        commit: 9ba415b
---
## Root Cause

Lipgloss's `Render()` always appends `\x1b[0m` (full ANSI reset) at the end of rendered text. When a badge is pre-rendered with lipgloss and concatenated into a title line, the badge's trailing reset cancels the outer `selectedTicketStyle`'s background for all text that follows. The previous fix (167b3f0) added `.Background("62")` to badge styles when selected, but this only colored the badge itself — the title text after the badge still lost its background.

## Fix

For selected cards only, bypass lipgloss and use raw ANSI escape sequences that change **only the foreground color** (no reset). The outer `selectedTicketStyle.Render()` sets the background once, and inline foreground-only changes preserve it throughout.

### Changes

**`internal/cli/tui/kanban/styles.go`** — Added 3 helper functions:
- `inlineFgColorChange(colorCode)` — returns raw `\x1b[38;5;{code}m` escape
- `typeBadgeColorCode(ticketType)` — maps ticket types to their 256-color codes
- `dueDateColorCode(overdue)` — returns color code for overdue (red) or due soon (orange)

**`internal/cli/tui/kanban/column.go`** — Modified `renderAllTickets()`:
- When card is selected: builds badge/due-date strings with inline ANSI foreground changes
- When card is not selected: unchanged lipgloss rendering

## Verification

- `make build` — compiles cleanly
- `make lint` — 0 issues
- `make test` — all unit tests pass