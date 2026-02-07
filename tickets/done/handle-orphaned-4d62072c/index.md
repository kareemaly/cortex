---
id: 4d62072c-96f4-43f0-9068-b1cfaaceb56b
title: Handle Orphaned Session Spawn Error with Resume/Fresh Dialog in TUI
type: ""
created: 2026-01-26T17:32:19.755574Z
updated: 2026-01-27T09:35:07.759316Z
---
## Problem

When spawning a ticket that has an orphaned session (tmux window was closed), the TUI shows a raw error:

```
Error: spawn: ticket 63e9ca04... in state orphaned: session was orphaned (tmux window closed). Use mode='resume' to continue or mode='fresh' to start over
```

There's no way for the user to act on this from the TUI — they have to manually re-run with the correct mode flag.

## Solution

Detect the orphaned session error and present a dialog with two options:

- **Resume** — calls spawn with `mode='resume'`
- **Fresh** — calls spawn with `mode='fresh'`

## Scope

- Parse the spawn error response to detect the orphaned state
- Show a modal/dialog in the TUI with the two options
- On selection, re-call the spawn API with the chosen mode
- This should work from both the kanban board view and ticket detail view

## Acceptance Criteria

- [ ] Orphaned session spawn error triggers a dialog instead of a raw error
- [ ] Dialog offers "Resume" and "Fresh start" options
- [ ] Selecting an option re-spawns with the correct mode
- [ ] Works from kanban board and ticket detail views