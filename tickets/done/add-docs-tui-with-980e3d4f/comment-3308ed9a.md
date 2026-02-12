---
id: 3308ed9a-3581-4dcd-aafe-72b71dbd4a44
author: claude
type: done
created: 2026-02-09T14:51:52.354211Z
---
## Completed: Add docs TUI with file explorer and markdown preview

### What was done
Implemented a read-only docs browsing TUI with a two-pane layout and wired it into the existing kanban TUI via tab switching.

### Files created (5)
- **`internal/cli/tui/views/model.go`** — Views wrapper hosting kanban + docs with tab/[/] switching. Routes WindowSizeMsg to both children (Height-1 for tab bar), keys to active child only, and all other messages (SSE, async) to both.
- **`internal/cli/tui/views/keys.go`** — View-switching key constants (tab, shift+tab, [, ])
- **`internal/cli/tui/docs/model.go`** — Two-pane docs browser: explorer (30%, tree view grouped by category) + preview (70%, glamour-rendered markdown with attribute bar). SSE live updates, doc caching, log viewer support.
- **`internal/cli/tui/docs/keys.go`** — Vim-style key constants and isKey() helper
- **`internal/cli/tui/docs/styles.go`** — Lipgloss styles: 8-color category cycle, tag pills, attribute bar, pane headers with focus indicator

### Files modified (2)
- **`cmd/cortex/commands/kanban.go`** — Changed to launch `views.New()` instead of `kanban.New()`
- **`CLAUDE.md`** — Updated CLI commands table and Key Paths for new TUI structure

### Verification
- `make build` — clean compilation
- `make lint` — 0 issues
- `make test` — all tests pass
- Pre-push hooks passed
- Merged to main and pushed