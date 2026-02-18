---
id: e903234a-df50-4fcb-9fe3-fdbbcbef0581
title: Add session kill keybinding (x) to dashboard
type: work
tags:
    - tui
    - session
created: 2026-02-18T07:08:48.02127Z
updated: 2026-02-18T07:25:42.767269Z
---
## Problem

Orphaned sessions (tmux window gone but session record remains) clutter the dashboard and there's no way to clean them up from the TUI. Active sessions also can't be terminated from the dashboard.

## Requirements

- Add `x` keybinding to the dashboard TUI
- When pressed on a session row, call the existing `DELETE /sessions/{id}` API via the SDK's `KillSession(id)` method
- **Orphaned sessions**: kill immediately, no confirmation needed
- **Active sessions**: show a confirmation prompt before killing (this terminates a running agent)
- Works for both architect and ticket agent sessions
- Dashboard should refresh after kill (the API emits a `SessionEnded` event which should trigger SSE update)

## Acceptance Criteria

- Pressing `x` on an orphaned session row removes it immediately
- Pressing `x` on an active session row shows confirmation; confirming kills it, cancelling does nothing
- The session disappears from the dashboard after being killed
- `x` on a project row (not a session row) does nothing or is ignored
- Help text / key hints updated to include `x`