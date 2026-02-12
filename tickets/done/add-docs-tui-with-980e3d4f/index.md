---
id: 980e3d4f-b5a4-4061-8d2d-39785593f7cc
title: Add docs TUI with file explorer and markdown preview
type: work
tags:
    - tui
    - docs
    - kanban
created: 2026-02-09T14:33:56.640574Z
updated: 2026-02-09T14:51:52.355922Z
---
## Goal

Add a docs browsing TUI with a two-pane layout (file explorer + markdown preview), and wire it into the existing kanban TUI with tab switching via `[`/`]`/`tab`/`shift-tab`.

## Design

### Layout: Two-Pane Split (30/70)

```
┌─ Docs ──────────────────────────────────────────────────────────────┐
│ ┌─ Explorer (30%) ──────┐ ┌─ Preview (70%) ─────────────────────┐  │
│ │                        │ │                                     │  │
│ │  📁 decisions          │ │  # API Cleanup Summary              │  │
│ │    ├─ api-cleanup-sum  │ │                                     │  │
│ │    └─ auth-approach    │ │  Centralized daemon URL across all  │  │
│ │  📁 findings           │ │  clients. Deduplicated response     │  │
│ │    └─ audit-results    │ │  types into internal/types...       │  │
│ │  📁 specs              │ │                                     │  │
│ │  ▸ 📁 sessions (4)     │ │                                     │  │
│ │                        │ │                                     │  │
│ │                        │ │─────────────────────────────────────│  │
│ │                        │ │ category: decisions  │ 2026-02-08   │  │
│ │                        │ │ tags: api, cleanup   │ ref: ticket:6│  │
│ └────────────────────────┘ └─────────────────────────────────────┘  │
│ [tab] Kanban  j/k navigate  h/l pane  q quit                       │
└─────────────────────────────────────────────────────────────────────┘
```

### Left Pane — File Explorer

- Tree view grouped by category (collapsible folders)
- Docs listed under their category, sorted by created descending
- `j/k` to navigate, `enter` or `l` to expand/collapse folders
- Highlighting a doc auto-updates the right pane preview
- Empty categories hidden

### Right Pane — Preview

Two sub-regions:
1. **Markdown body** (top ~85%) — glamour-rendered, scrollable viewport
2. **Attribute bar** (bottom 2-3 lines) — category, tags as colored pills/badges, created/updated dates, references. Compact, styled with lipgloss

When nothing is selected, show a subtle empty state.

### Navigation

| Key | Action |
|-----|--------|
| `j/k` | Move up/down in active pane |
| `h/l` | Switch focus between explorer ↔ preview |
| `enter` | Expand/collapse category in explorer |
| `gg` / `G` | Top / bottom |
| `ctrl+u/d` | Page scroll in preview |
| `r` | Refresh |
| `q` | Quit |
| `!` | Toggle log viewer |

### View Switching (Kanban ↔ Docs)

**Wrapper model approach**: Create a thin top-level `views` model that wraps both the kanban and docs models. It intercepts `[`/`]`/`tab`/`shift-tab` to toggle which child is active and rendered. Both models stay alive so scroll position and loaded data are preserved when switching.

- `tab` / `]` → next view
- `shift-tab` / `[` → previous view
- The `cortex kanban` command launches this wrapper instead of kanban directly
- Status bar shows view indicator (e.g. `[Kanban] Docs` vs `Kanban [Docs]`)

## Scope — v1 Read-Only

- Browse and preview only, no doc actions (create/delete/move)
- No search/filter — just tree navigation
- SSE integration for live updates (follow existing pattern)

## Implementation

### New Files

| File | Purpose |
|------|---------|
| `internal/cli/tui/docs/model.go` | Docs TUI main model (explorer + preview) |
| `internal/cli/tui/docs/keys.go` | Keybindings |
| `internal/cli/tui/docs/styles.go` | Lipgloss styles (category colors, tag pills, attribute bar) |
| `internal/cli/tui/views/model.go` | Wrapper model that holds kanban + docs, handles tab switching |
| `internal/cli/tui/views/keys.go` | View switching keybindings |

### Modified Files

| File | Change |
|------|--------|
| `cmd/cortex/commands/kanban.go` | Launch `views.New(client, logBuf)` wrapper instead of `kanban.New()` directly |
| `internal/cli/tui/kanban/model.go` | Ensure it works as a child model (it should already, but verify `Init`/`Update`/`View` are clean) |

### Patterns to Follow

- Reuse `tuilog.Viewer` for log overlay (same `!` toggle pattern)
- Reuse SSE subscription pattern from kanban
- Use `glamour.NewTermRenderer` for markdown (same as ticket detail TUI)
- Use `viewport.Model` from bubbles for the preview scroll
- Vim-style navigation (j/k, gg, G, ctrl+u/d)
- Standard status bar layout (2 lines: message + help text)

### Data Flow

- `client.ListDocs("", "", "")` → get all docs with summaries
- Group by category client-side, sort each group by created desc
- On doc highlight: `client.FindDocByID(id)` → full doc with body + comments
- Cache fetched doc bodies to avoid re-fetching on re-highlight

## Notes

- The wrapper model is the key architectural piece — get that right and each child view stays cleanly independent
- Keep the kanban model interface unchanged so it's a non-breaking refactor
- Log viewer should work in both views (wrapper passes `!` through to whichever is active)