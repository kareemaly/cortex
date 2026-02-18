---
id: 84757c3a-edbe-45ab-9bb8-cd0bfc4df092
author: claude
type: comment
created: 2026-02-12T15:02:27.645944Z
---
## Root Cause Analysis

**Root Cause:** Pre-rendered ANSI escape codes in the type badge prevent the selection background from being applied uniformly.

**Code flow:**
1. `column.go:120` — The type badge is rendered with `typeBadgeStyle(t.Type).Render("[" + t.Type + "] ")`, which outputs a string containing ANSI escape codes for the foreground color only (no background).
2. `column.go:142` — This pre-rendered string is concatenated into the first title line: `wrappedTitle[0] = typeBadge + wrappedTitle[0]`
3. `column.go:156` — The entire line (badge + title) is passed to `selectedTicketStyle.Width(width - 2).Render(line)`, which attempts to apply background color `62` to the whole line.

**Why it fails:** Lipgloss cannot retroactively apply a background color to text that already has its own ANSI escape sequences. The badge's escape codes create a "styled island" within the string where the outer style's background doesn't penetrate. The result: the badge text appears without the selection background while the rest of the title is highlighted.

**Same issue exists for:** Due date indicators (`[OVERDUE]`/`[DUE SOON]`) rendered at lines 128-131, which are also pre-styled and appended to the first title line. These only appear in the backlog column but have the same visual break.