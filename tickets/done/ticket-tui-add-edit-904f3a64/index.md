---
id: 904f3a64-35db-402b-8a0c-2d4744ead2ad
title: 'Ticket TUI: add edit shortcut, kanban opens ticket as tmux popup'
type: work
tags:
    - tui
    - cleanup
created: 2026-02-10T10:08:42.420338Z
updated: 2026-02-10T10:27:55.345644Z
---
## Changes

**Ticket TUI (`cortex show`):**
- Add `e` shortcut that opens the ticket's file in `$EDITOR` via a tmux popup

**Kanban view:**
- When selecting a ticket in kanban, open it as a tmux popup running `cortex show <id>` instead of the current nested/embedded ticket detail view
- Remove the inline ticket detail view from the kanban TUI

## Acceptance Criteria

- `e` in ticket TUI opens a tmux popup with `$EDITOR` on the ticket file
- Selecting a ticket in kanban launches `cortex show` in a tmux popup
- No more nested ticket view inside kanban
- Tmux is a required dependency — no need for non-tmux fallback