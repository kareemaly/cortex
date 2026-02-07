---
id: a8f8d675-0755-454d-8592-3c46a7fc90cf
title: Add resume/fresh mode selection for orphaned architect in dashboard TUI
type: work
tags:
    - tui
    - dashboard
    - architect
    - ux
created: 2026-02-07T12:48:29.549853Z
updated: 2026-02-07T12:54:55.361511Z
---
## Problem

When the architect session is orphaned, the dashboard TUI shows the orphaned state visually (hollow circle icon, `[arch: orphaned]` badge) but doesn't provide a way to resume or start fresh. Pressing `[s]` calls `spawnArchitect("")` with an empty mode, which triggers a 409 from the API — but the TUI doesn't handle this error or offer mode selection.

The CLI handles this correctly by catching `IsOrphanedSession()` and printing options for `--mode fresh` and `--mode resume`.

## Requirements

When the architect is orphaned, the TUI should let the user choose between **resume** and **fresh** before spawning. Two approaches (implementer should pick the simpler one):

**Option A — Inline prompt on [s]:** When `[s]` is pressed and architect state is orphaned, show a status-bar prompt like `"Orphaned session: [r]esume / [f]resh / [esc] cancel"` and wait for the next keypress to dispatch the correct mode.

**Option B — Handle 409 after spawn attempt:** If `spawnArchitect("")` returns `session_orphaned` error, show the same prompt and retry with the selected mode.

Option A is preferred since we already know the state before the user presses [s].

## Key files

- `internal/cli/tui/dashboard/model.go` — spawn handler (~line 1017), focus handler (~line 465-499), status rendering
- `internal/cli/tui/dashboard/keys.go` — keybindings and help text
- `internal/cli/sdk/client.go` — `SpawnArchitect(mode)` and `IsOrphanedSession()`

## Acceptance criteria

1. When architect is orphaned and user presses [s], they are prompted to choose resume or fresh
2. Selecting resume calls `spawnArchitect("resume")`, fresh calls `spawnArchitect("fresh")`
3. Escape cancels the prompt and returns to normal state
4. Help text or status message clearly communicates the available options when orphaned
5. The orphaned focus handler (pressing enter/f on orphaned architect row) should also trigger the mode selection instead of just showing a status message