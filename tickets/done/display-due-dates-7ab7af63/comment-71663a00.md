---
id: 71663a00-0dc8-4755-bc9f-964a9d96e4cc
author: claude
type: comment
created: 2026-02-07T11:20:20.646418Z
---
## Root Cause Analysis

**Investigated three areas for due date display gaps:**

### 1. Architect Context Injection — GAP FOUND
- **File**: `internal/core/spawn/spawn.go:619-629`
- **Root cause**: The `writeSection()` function formats tickets as `- [ID] Title (updated: date)` but **ignores the `Due` field** entirely, even though `TicketSummary.Due` is populated and available.
- The `Due *time.Time` field is present in `types.TicketSummary` and flows through correctly from the API — it's simply not rendered in the prompt text.

### 2. Kanban TUI — ALREADY IMPLEMENTED ✅
- **File**: `internal/cli/tui/kanban/column.go:123-132`
- Due dates are already displayed with `[OVERDUE]` (red, color 196) and `[DUE SOON]` (orange, color 214) badges appended to the ticket title line.

### 3. Ticket Detail TUI — ALREADY IMPLEMENTED ✅
- **File**: `internal/cli/tui/ticket/model.go:1108-1120`
- Due dates are already displayed in the attributes panel with color-coded urgency (red for overdue, orange for due soon, neutral for normal).

**Conclusion**: Only the architect context injection needs fixing. The TUI components already satisfy the acceptance criteria.