---
id: b213cc6d-e9e1-4a47-9bbd-b6517cdd7389
title: 'Ticket TUI: Split Panel Layout with Description and Sidebar'
type: ""
created: 2026-01-27T15:19:30.493441Z
updated: 2026-01-27T15:44:22.791187Z
---
## Summary

Redesign the ticket detail TUI (`cortex show`) with a Jira-style 70/30 split layout. Left panel shows the scrollable description, right panel shows metadata sidebar.

## Layout

```
┌──────────────────────────────────────────────────────────────────────────┐
│  [a1b2c3d4] Fix spawn bug                                    progress   │  ← 5% header
├───────────────────────────────────────────────┬──────────────────────────┤
│                                               │  DETAILS                 │
│  ## Description                               │  Created  Jan 27, 14:05  │
│                                               │  Updated  Jan 27, 15:30  │
│  The spawn operation fails when the tmux      │  Progress Jan 27, 14:10  │
│  session target uses prefix matching...       │                          │
│                                               │  SESSION                 │
│  ### Steps to reproduce                       │  Agent    claude          │
│  1. Open the kanban                           │  Status   in_progress     │
│  2. Select a backlog ticket                   │  Tool     Bash            │
│  3. Press 's' to spawn                        │  Window   fix-spawn-bug   │
│                                               │  Started  Jan 27, 14:10  │
│                                               │                          │
│                                               │  REVIEWS (1)             │
│                                               │    cortex1 · Fix tmux..  │
│                                               │                          │
│                                               │  COMMENTS (3)            │
│                                               │    progress · Found...   │
│                                               │    decision · Switch..   │
│                                               │    blocker · Need...     │
│                                               │                          │
│  [LEFT FOCUSED]                               │                          │
├───────────────────────────────────────────────┴──────────────────────────┤
│  [h/l] panel  [j/k] scroll  [r]efresh  [f]ocus  [ga] architect    75%  │
└──────────────────────────────────────────────────────────────────────────┘
```

## Changes

### Split layout
- Left panel (70%): scrollable viewport with glamour-rendered description/body
- Right panel (30%): static metadata sidebar — details, session, reviews list, comments list
- Header (5%): ticket ID + title + status badge (unchanged)
- Help bar: context-sensitive based on focused panel

### Panel focus with h/l
- `h` focuses left panel (description scrolling)
- `l` focuses right panel (list navigation — implemented in follow-up ticket)
- Visual indicator showing which panel is focused (e.g., border highlight or label)
- Default focus: left panel

### Left panel behavior
- Scrollable viewport with `j/k`, `gg/G`, `Ctrl+D/U`
- Glamour markdown rendering of ticket body
- Scroll percentage shown in help bar

### Right panel rendering
- DETAILS section: Created, Updated, Progress, Reviewed, Done timestamps
- SESSION section (if exists): Agent, Status, Tool, Window, Started, Ended
- REVIEWS section: one-liner per review (repo path + summary/title)
- COMMENTS section: one-liner per comment (type + title)
- Review repo path should be relative to project folder; omit if same as project

### Narrow terminal fallback
- Below ~100 columns, collapse to single-column stacked layout (current behavior)

## Files
- `internal/cli/tui/ticket/model.go` — restructure View(), add split layout, panel focus state
- `internal/cli/tui/ticket/styles.go` — add panel styles, focus indicators
- `internal/cli/tui/ticket/keys.go` — add h/l panel switching, update help text