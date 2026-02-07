---
id: 612e580d-e3eb-420b-8f54-2922952b4782
title: 'Ticket TUI: Header/Comment Padding, Comment Highlight, Bracket Shortcuts, and 3-Line Markdown Previews'
type: work
created: 2026-01-28T10:43:24.445548Z
updated: 2026-01-28T13:16:23.449824Z
---
## Summary

A set of visual and interaction improvements to the ticket detail TUI (`internal/cli/tui/ticket/`). These changes improve readability, navigation, and comment preview quality.

## Requirements

### 1. Header vertical padding
- Add top and bottom padding to the header bar (the row containing ticket ID, title, and status badge)
- Currently uses `Padding(0, 1)` — needs vertical padding added (e.g., `Padding(1, 1)` or similar)
- File: `styles.go` → `headerStyle`

### 2. Comment row padding and full-row highlight
- Add top and bottom padding to each comment row in the comment list
- When a comment is selected (cursor hover), the **entire comment row** must be highlighted — not just the badge/type. Currently `commentSelectedStyle` applies a background but may not visually cover the full row including padding
- Ensure the highlight covers the full width and height of the padded comment row
- Files: `styles.go` → `commentSelectedStyle`, `model.go` → `renderCommentLine()`

### 3. Bracket shortcuts for panel switching
- Add `[` and `]` as additional keybindings to switch focus between body (Row 0) and comments (Row 1)
- Same behavior as existing `Tab` / `Shift+Tab` toggle
- Update help text to reflect the new shortcuts
- Files: `keys.go`, `model.go` → `handleKeyMsg()`

### 4. 3-line markdown-rendered comment previews
- Each comment in the list should show up to **3 lines** of content (currently shows 1 line of plain text)
- Render comment content as styled markdown using `glamour` (already used for body and modal)
- **Truncation strategy**: Render the full markdown first with glamour, then truncate the **rendered output** to 3 lines. This avoids breaking markdown syntax mid-render — we're cutting styled/ANSI text, not raw markdown source
- Keep the existing badge and time-ago elements in the comment row layout
- Ensure the 3-line preview respects the available width and doesn't overflow

## Acceptance Criteria

- [ ] Header has visible vertical padding (top and bottom)
- [ ] Each comment row has vertical padding
- [ ] Selecting a comment highlights the entire row (background covers full width and height)
- [ ] `[` and `]` keys toggle panel focus, same as Tab
- [ ] Help text includes bracket shortcuts
- [ ] Comment previews show up to 3 lines of glamour-rendered markdown
- [ ] Markdown truncation does not produce broken/garbled output
- [ ] No visual overflow or layout breakage at various terminal widths