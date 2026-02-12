---
id: 337397f1-2364-40a0-86f9-e9cff882bc81
author: claude
type: review_requested
created: 2026-02-10T10:22:54.772882Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/692ef108-ef78-4642-80c8-c07e60401493
---
## Docs TUI: Fix Explorer Highlight, Multiline Titles, Add Edit Shortcut

### Changes Made

**7 files modified:**

1. **`internal/cli/tui/docs/keys.go`** — Removed `KeyEnter`, added `KeyE` constant, updated help text to show `[e]dit` instead of `[enter] expand`

2. **`internal/cli/tui/docs/styles.go`** — Replaced `selectedStyle` (full-row background highlight) with a text-only bold white style + new `selectedIndicator` style using accent color for the `▎` left-border indicator

3. **`internal/cli/tui/docs/model.go`** — Core TUI changes:
   - Replaced `enter` key handler with `e` key handler calling `editSelectedDoc()`
   - Added `DocEditMsg` message type and handler in `Update()`
   - Added `editSelectedDoc()` method that calls `client.OpenDocInEditor(id)`
   - Changed `renderExplorer()` to use `▎` left-border indicator instead of full-row background color (focused = accent color, unfocused = dim gray)
   - Changed `renderTreeItem()` to use `lipgloss.NewStyle().Width(maxTitle).Render(title)` for natural wrapping instead of truncation with `...`
   - Changed viewport scroll logic from cursor-index-based to line-offset-based tracking (`itemStartLines` array) to correctly handle multi-line items

4. **`internal/docs/store.go`** — Added `GetFilePath(id)` method that uses existing `findEntityDir()` to return the filesystem path to a doc's `index.md`

5. **`internal/daemon/api/docs.go`** — Added `Edit()` HTTP handler (POST /docs/{id}/edit) that resolves `$EDITOR` (fallback `vi`), gets tmux session name from project config, and opens a tmux popup with the editor

6. **`internal/daemon/api/server.go`** — Registered `r.Post("/{id}/edit", docHandlers.Edit)` route in the docs route group

7. **`internal/cli/sdk/client.go`** — Added `OpenDocInEditor(id)` method that POSTs to `/docs/{id}/edit`

### Verification
- `make build` — passes
- `make lint` — 0 issues
- `make test` — all tests pass