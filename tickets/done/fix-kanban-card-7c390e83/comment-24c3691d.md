---
id: 24c3691d-43b2-4f54-a8e1-bb870c258bf1
author: claude
type: done
created: 2026-02-13T08:38:25.217074Z
---
## Root Cause

Lipgloss's `Render()` appends `\x1b[0m` (full ANSI reset) at the end of every rendered string. When type badges (`[work]`, `[debug]`, etc.) and due date indicators (`[OVERDUE]`, `[DUE SOON]`) were pre-rendered with lipgloss and concatenated into the title line, the badge's trailing reset canceled the outer `selectedTicketStyle`'s background for all subsequent text. The previous fix (commit 167b3f0) added `.Background("62")` to badge styles when selected, but this only colored the badge itself — the title text after the badge still lost its background styling.

## Resolution

For selected cards only, bypassed lipgloss and used raw ANSI escape sequences (`\x1b[38;5;{code}m`) that change only the foreground color without resetting other attributes. The outer `selectedTicketStyle.Render()` sets the background once, and inline foreground-only changes preserve it across the entire line — badge, title, and due date indicator all share a uniform background.

## Files Changed

- `internal/cli/tui/kanban/styles.go` — Added `inlineFgColorChange()`, `typeBadgeColorCode()`, `dueDateColorCode()` helper functions
- `internal/cli/tui/kanban/column.go` — Modified `renderAllTickets()` to use inline ANSI for badges/due dates on selected cards; non-selected cards unchanged

## Verification

- `make build` — compiles cleanly
- `make lint` — 0 issues
- `make test` — all unit tests pass

Commit: 9ba415b