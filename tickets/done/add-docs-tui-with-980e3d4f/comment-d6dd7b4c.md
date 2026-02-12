---
id: d6dd7b4c-7b29-46a0-9427-fff7b9b23e8b
author: claude
type: review_requested
created: 2026-02-09T14:46:36.711884Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/9f2f589b-6523-44ee-a264-dd56a7bf4af4
        commit: d7d7b1d
---
## Summary

Added a docs browsing TUI with two-pane layout (file explorer + markdown preview) and wired it into the existing kanban TUI via tab switching.

### New Files (5)

**`internal/cli/tui/views/model.go`** — Thin top-level wrapper (`views.Model`) that hosts both `kanban.Model` and `docs.Model`. Handles view switching with `tab`/`shift-tab`/`[`/`]`. Routes `tea.WindowSizeMsg` to both children (with `Height-1` for tab bar), routes key messages to active child only, and routes all other messages (SSE events, async results) to both children.

**`internal/cli/tui/views/keys.go`** — View-switching key constants and helpers.

**`internal/cli/tui/docs/model.go`** — Two-pane docs browser:
- **Explorer (left, 30%)**: Tree view grouped by category (collapsible), sorted alphabetically with docs sorted by created desc. Uses `viewport.Model` for scrolling. Cursor navigation with j/k, expand/collapse with enter/l.
- **Preview (right, 70%)**: Glamour-rendered markdown body in a scrollable viewport, with an attribute bar at the bottom showing category badge, tag pills, dates, and references.
- Data flow: `ListDocs` on init → build tree → `GetDoc` on cursor move (cached by ID) → glamour render
- SSE integration for live updates (same pattern as kanban)
- Log viewer support (! toggle, same as kanban)

**`internal/cli/tui/docs/keys.go`** — Key constants and `isKey()` helper (same pattern as kanban).

**`internal/cli/tui/docs/styles.go`** — Lipgloss styles: category colors (8-color cycle), selected item, tree connectors, attribute bar styles (category badge, tag pills, dates, refs), pane headers with active focus indicator.

### Modified Files (1)

**`cmd/cortex/commands/kanban.go`** — Changed from `kanban.New(client, logBuf)` to `views.New(client, logBuf)` so `cortex kanban` launches the wrapper with both views.

### Verification
- `make build` — compiles cleanly
- `make lint` — 0 issues
- `make test` — all tests pass