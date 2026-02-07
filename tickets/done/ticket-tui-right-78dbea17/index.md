---
id: 78dbea17-ad2d-4145-97e6-5e9c84409887
title: 'Ticket TUI: Right Panel Navigation and Detail Modals'
type: ""
created: 2026-01-27T15:19:30.64136Z
updated: 2026-01-28T05:15:00.998784Z
---
## Summary

Add interactive navigation to the ticket TUI right panel. Reviews and comments become a navigable cursor list. Opening an item shows a scrollable detail modal with context-specific actions.

## Depends On

- Ticket TUI: Split Panel Layout with Description and Sidebar

## Right Panel Navigation

When right panel is focused (`l`):
- `j/k` moves cursor through reviews + comments as a flat list
- `gg` jumps to first item, `G` jumps to last
- Selected item is highlighted
- `o` or `Enter` opens the detail modal for the selected item

### Item rendering (one-liner each)
- **Review**: `{relative_repo_path} · {summary/title}` — repo path relative to project folder, omitted if same as project
- **Comment**: `{type} · {title}` — falls back to first line of content if no title

## Detail Modal

Centered overlay modal (~60% width, ~70% height), rendered on top of the split layout.

### Comment modal
```
┌───────────────────────────────────────┐
│  progress · Jan 27, 14:20             │
│───────────────────────────────────────│
│                                       │
│  Found the root cause in session.go.  │
│  The prefix matching logic is too     │
│  greedy and matches multiple sessions │
│  when the name is a prefix of another.│
│                                       │
│  ## Solution                          │
│  Use exact match with `=` suffix in   │
│  the tmux target specification.       │
│                                       │
│  [Esc/q] close          [j/k] scroll  │
└───────────────────────────────────────┘
```

### Review modal (with actions)
```
┌───────────────────────────────────────┐
│  Review Request · 2m ago              │
│───────────────────────────────────────│
│  Repo: src/services (relative path)   │
│                                       │
│  Changed session target resolution    │
│  to use exact match. Prevents         │
│  ambiguous target errors when         │
│  session names share prefixes.        │
│                                       │
│  [Esc/q] close  [a]pprove  [x]reject  │
└───────────────────────────────────────┘
```

## Modal behavior
- Content rendered with glamour markdown
- `j/k` scrolls modal content (uses its own viewport)
- `Esc` or `q` closes the modal
- Review modals: `a` approves, `x` rejects (calls existing API)
- Modal takes input priority (all other keys ignored while open)

## Navigation summary

| Context | Key | Action |
|---------|-----|--------|
| Any | `h` | Focus left panel |
| Any | `l` | Focus right panel |
| Left focused | `j/k` | Scroll description |
| Left focused | `gg/G` | Jump top/bottom |
| Right focused | `j/k` | Move cursor through items |
| Right focused | `gg/G` | First/last item |
| Right focused | `o/Enter` | Open detail modal |
| Modal open | `j/k` | Scroll modal content |
| Modal open | `Esc/q` | Close modal |
| Modal (review) | `a` | Approve |
| Modal (review) | `x` | Reject |

## Files
- `internal/cli/tui/ticket/model.go` — right panel cursor state, modal state, modal rendering
- `internal/cli/tui/ticket/styles.go` — modal styles, cursor highlight
- `internal/cli/tui/ticket/keys.go` — right panel keys, modal keys