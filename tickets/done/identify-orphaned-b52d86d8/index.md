---
id: b52d86d8-9063-42c1-963a-c62be6aa7307
title: Identify orphaned sessions and allow deletion in TUIs
type: work
created: 2026-02-05T11:31:49.664563Z
updated: 2026-02-05T11:54:09.784662Z
---
Dashboard and Kanban TUIs need to clearly identify orphaned sessions (sessions that no longer have a tmux window) and allow manual deletion.

**Requirements:**
1. Visually distinguish orphaned sessions (different color/icon/label)
2. Add `[D]` (uppercase) shortcut to delete orphaned sessions
3. Show confirmation before deletion

**Shortcut Analysis:**
- `[D]` recommended - Vim idiom for destructive actions, no conflicts, clear mnemonic
- Current orphan modal has: `[r]esume`, `[f]resh`, `[c]ancel`
- New modal options: `[r]esume  [f]resh  [D]elete  [c]ancel`

**Locations:**
- `internal/cli/tui/kanban/` - orphan modal in `model.go:handleOrphanModalKey()`
- `internal/cli/tui/ticket/` - orphan modal in `model.go:handleOrphanModalKey()`
- `internal/cli/tui/dashboard/` - session display