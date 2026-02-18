---
id: b862c21c-8dd6-45ec-bc36-e89aa4332fa0
author: claude
type: done
created: 2026-02-13T08:31:52.219216Z
---
## Fix: Explorer selection highlight not visually working

### Problem
The previous approach (commit `f0df444`) wrapped pre-styled lines with `selectedStyle.Render(line)`, but lipgloss's outer `Render()` cannot override inner ANSI escape sequences from styles like `categoryStyle`, `treeConnector`, `configItemStyle`, and `ejectedBadgeStyle`. The accent color only applied to unstyled text segments — effectively nothing.

### Solution
Passed selection state (`selected bool, unfocused bool`) into `renderTreeItem` (docs) and `renderListItem` (config) so they apply the correct foreground color **directly** to each individual text segment.

- **Focused selection**: accent color (#62) + bold on all segments
- **Unfocused selection**: gray (245) on all segments  
- **Unselected**: original per-segment styling preserved

### Files changed
1. `internal/cli/tui/docs/model.go` — `renderTreeItem` accepts selection state, `renderExplorer` passes it
2. `internal/cli/tui/config/model.go` — `renderListItem` accepts selection state, `renderExplorer` passes it
3. `internal/cli/tui/docs/styles.go` — Removed unused `selectedStyle`/`unfocusedSelectedStyle`
4. `internal/cli/tui/config/styles.go` — Same cleanup

### Verification
- `make build` ✓
- `make lint` — 0 issues ✓
- `make test` — all tests pass ✓
- Pre-push hooks passed ✓
- Pushed to origin/main as commit `7e5819d`