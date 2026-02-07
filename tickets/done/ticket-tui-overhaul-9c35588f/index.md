---
id: 9c35588f-d8ee-4e8b-a88b-0bff0ef8491d
title: 'Ticket TUI Overhaul: Row-Based Layout with Unified Comment List'
type: ""
created: 2026-01-28T09:35:31.000965Z
updated: 2026-01-28T10:31:12.248117Z
---
## Summary

Redesign the ticket detail TUI from the current 70/30 horizontal split to a row-based layout with two vertically stacked sections that dynamically resize based on focus. Render all comments (including reviews) in a unified list with type-based visual styling and strip-and-truncate markdown previews.

**Depends on**: "Simplify Comment Model" ticket (needs the unified comment model with 4 types and no separate reviews)

## Current State

- Split layout: 70% left (body), 30% right (metadata + reviews + comments)
- h/l switches panel focus
- Reviews and comments rendered separately in sidebar
- Comments show type badge + title
- o/Enter opens detail modal

## Target State

### Layout

```
┌──────────────────────────────────────────────────┐
│ [ID]  Title                           [status]   │  ← Header (fixed)
├───────────────────────────────┬──────────────────┤
│                               │  DETAILS         │
│  Ticket description           │  created: ...    │  ← Row 1
│  (markdown rendered)          │  type: work      │
│                               │  session: ...    │
├───────────────────────────────┴──────────────────┤
│ [blocker]  Agent can't find the config file af…  │
│ [comment]  Updated spawn logic to handle new t…  │  ← Row 2
│ [review]   All changes complete, ready for rev…  │
│ [done]     Session concluded. Implemented all …  │
└──────────────────────────────────────────────────┘
```

- **Header**: ticket ID, title, status badge (fixed height, always visible)
- **Row 1**: 70% width ticket description (markdown), 30% width attributes sidebar (dates, type, session info)
- **Row 2**: full-width unified comment list
- Focused row takes **70% of vertical space**, unfocused row takes 30%
- Default focus: Row 1

### Navigation

- `Tab` / `Shift+Tab` — switch focus between Row 1 and Row 2
- **Row 1 focused**: j/k scrolls ticket description
- **Row 2 focused**: j/k navigates between comments in the list
- `Enter` on a selected comment — opens full-screen detail modal with full markdown-rendered content, j/k scrolls within, Esc closes
- Existing shortcuts preserved: `r` refresh, `ga` architect, `s` spawn, `a` approve, `x` kill, `q` quit

### Comment list rendering

- Each comment is a single line: `[type_badge]  preview_text…`
- **Type badges** with distinct colors:
  - `[blocker]` — red
  - `[review]` — yellow
  - `[comment]` — gray/default
  - `[done]` — green
- **Preview text**: strip markdown syntax from first non-empty line, truncate to available width with ellipsis
  - Strip logic: trim leading `#`, `*`, `-`, `>`, whitespace from first non-empty line
  - Truncate with `…` at available terminal width minus badge width and padding
- Selected comment highlighted with background color

### Detail modal

- Full-screen overlay (existing modal pattern)
- Shows comment type badge + timestamp at top
- Full content rendered as markdown via Glamour
- If type is `review_requested` and action is `git_diff`: show repo_path and commit info
- j/k scrolls content, Esc closes
- Action shortcuts if applicable (e.g., `a` approve if review_requested)

## Acceptance Criteria

- [ ] Row-based layout with header, row1 (description + attributes), row2 (comments)
- [ ] Focused row takes 70% vertical space, unfocused takes 30%
- [ ] Tab/Shift+Tab switches focus between rows
- [ ] Row 1: j/k scrolls ticket description, 30% sidebar shows attributes
- [ ] Row 2: j/k navigates comment list
- [ ] Comments rendered as single-line previews with type badge and stripped/truncated markdown
- [ ] Strip logic handles common markdown prefixes (#, *, -, >, whitespace)
- [ ] Enter opens detail modal with full markdown-rendered comment content
- [ ] Modal shows action metadata (repo_path, commit) for review_requested comments
- [ ] All 4 comment types have distinct color-coded badges
- [ ] Existing shortcuts preserved (r, ga, s, a, x, q)
- [ ] SSE real-time updates still work
- [ ] Works in both narrow and wide terminals