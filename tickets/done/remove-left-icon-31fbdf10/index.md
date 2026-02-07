---
id: 31fbdf10-c044-4f52-b798-0562c5e428c2
title: Remove Left Icon Prefix from Ticket Cards
type: ""
created: 2026-01-27T09:58:26.414508Z
updated: 2026-01-27T10:01:46.332177Z
---
## Problem

The ticket card title still has left-side icon prefixes that push content to the right:

- **Unselected with active session:** `● ` (agent status icon) on title line 1
- **Selected/focused:** `> ` on title line 1

Since the agent status is now shown on the bottom metadata line (`● ToolName · Jan 27`), the top-left icon is redundant. The `> ` focus indicator is also unnecessary visual noise.

### Current
```
● MCP Spawn Should
  Delegate to Daemon
  HTTP API
  ● Read · Jan 27
```

### Expected
```
  MCP Spawn Should
  Delegate to Daemon
  HTTP API
  ● Read · Jan 27
```

The agent status icon should only appear on the metadata line — nowhere else on the card.

## Scope

- **`internal/cli/tui/kanban/column.go`**:
  - Remove `> ` prefix for selected/focused title line 1 (line ~188-189) — use plain `  ` instead
  - Remove `● ` icon prefix for unselected active-session title line 1 (lines ~204-207) — use plain `  ` instead
  - All title lines should use the same `  ` indent regardless of state

## Acceptance Criteria

- [ ] No icon prefix on title line 1 in any state (selected, unselected, active session, no session)
- [ ] All title lines use uniform `  ` indent
- [ ] Agent status still renders on the bottom metadata line next to the date
- [ ] Selected ticket is still visually distinct (via highlight style, not prefix icon)