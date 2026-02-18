---
id: b2801a6d-7f1a-4a51-8f8f-6df7ba06682d
author: claude
type: review_requested
created: 2026-02-12T15:04:57.265381Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
        commit: 167b3f0
---
**Root cause:** Type badge and due date indicator styles only set foreground color. When pre-rendered into ANSI strings, the outer `selectedTicketStyle` background (color "62") cannot override them, creating unstyled "islands" in the highlight.

**Fix (internal/cli/tui/kanban/column.go):**
- Type badge: When the card is selected (`i == c.cursor && isActive`), add `.Background(lipgloss.Color("62")).Bold(true)` to the badge style before rendering.
- Due date indicators (`[OVERDUE]`, `[DUE SOON]`): Same treatment — conditionally add the selection background to `overdueStyle` / `dueSoonStyle`.

**Verification:** `make build` (compiles), `make lint` (0 issues), `make test` (all pass).