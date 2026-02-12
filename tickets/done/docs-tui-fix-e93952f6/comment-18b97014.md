---
id: 18b97014-b8db-4845-a596-07de88873e8e
author: claude
type: done
created: 2026-02-10T10:24:05.918747Z
---
## Docs TUI: Fix Explorer Highlight, Multiline Titles, Add Edit Shortcut

### Changes (7 files, commit 1512e14)

**Visual fixes:**
- Replaced broken full-row background highlight with `▎` left-border indicator (accent color when focused, dim gray when unfocused) in `styles.go` and `model.go`
- Changed long doc titles from truncation (`...`) to natural multiline wrapping using `lipgloss.NewStyle().Width()` in `renderTreeItem()`
- Updated viewport scroll logic from cursor-index-based to line-offset-based (`itemStartLines` array) to correctly handle variable-height items

**Keybinding changes:**
- Removed unused `enter` key binding (category toggle already handled by `l` key)
- Added `e` shortcut to open docs in `$EDITOR` via tmux popup

**New endpoint stack:**
- `docs.Store.GetFilePath(id)` — resolves doc ID to filesystem path
- `POST /docs/{id}/edit` API handler — resolves `$EDITOR`, opens tmux popup
- `sdk.Client.OpenDocInEditor(id)` — SDK method for the new endpoint

All verification passed: `make build`, `make lint` (0 issues), `make test` (all pass), pre-push hooks passed.