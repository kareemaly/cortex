---
id: 026cc665-e9ed-48d8-878c-807b7b47165e
title: Redesign Kanban Ticket Card Layout
type: ""
created: 2026-01-27T09:32:43.466033Z
updated: 2026-01-27T09:45:51.603861Z
---
## Problem

The current ticket card layout in the Kanban TUI has two issues:

1. **Agent status steals title space** — The agent prefix (`● TodoWrit `) on the first line of the title consumes ~12 characters, pushing title words to overflow lines. The title width is `column_width - 6`, but the agent prefix takes additional space only on line 1, causing uneven wrapping.

2. **Focused state hides agent status** — When a ticket is focused/selected, the agent status icon and tool name disappear entirely, replaced by just `> `. The user loses visibility into what the agent is doing.

## Solution

Move the agent status (icon + tool name) from the title's first-line prefix to the bottom metadata line, alongside the date. Use a `·` separator between them.

### Before (current)
```
● TodoWrit Split agent_args
  into
  Architect and Ticket
  Session Args
  Jan 27
```

### After (redesign)
```
  Split agent_args into
  Architect and Ticket
  Session Args
  ● TodoWrit · Jan 27
```

### Focused state (after)
```
> Split agent_args into
  Architect and Ticket
  Session Args
  ● TodoWrit · Jan 27
```

## Scope

- **`internal/cli/tui/kanban/column.go`** — Main changes:
  - Remove agent status from the title first-line prefix in both selected and unselected states
  - Move agent status rendering to the date/metadata line
  - Format as `● ToolName · Jan 27` when there's an active session, plain `Jan 27` otherwise
  - Keep `> ` prefix for focused state on title line 1
  - Keep `● ` (icon only, no tool) or `  ` for unselected title line 1 prefix

- **`internal/cli/tui/kanban/styles.go`** — May need a style for the combined metadata line

## Acceptance Criteria

- [ ] Agent status (icon + tool) renders on the bottom metadata line next to the date
- [ ] Title gets full width on all lines (no agent tool name stealing space)
- [ ] Focused/selected ticket still shows agent status on the metadata line
- [ ] Tickets without active sessions show just the date on the metadata line
- [ ] No regression in scroll indicators, column layout, or ticket height