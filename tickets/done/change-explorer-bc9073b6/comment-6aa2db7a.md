---
id: 6aa2db7a-dbcb-4d88-ba59-605587af0c09
author: claude
type: review_requested
created: 2026-02-13T08:27:04.703406Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Fix: Explorer selection highlight not visually working

### Problem
The previous commit (`f0df444`) wrapped pre-styled lines with `selectedStyle.Render(line)`, but lipgloss outer `Render()` cannot override inner ANSI escape sequences from `categoryStyle`, `treeConnector`, `configItemStyle`, `ejectedBadgeStyle`, etc. — so the accent color only applied to unstyled text segments (almost nothing).

### Solution
Pass selection state (`selected bool, unfocused bool`) into `renderTreeItem` (docs) and `renderListItem` (config) so they apply the correct foreground color **directly** to each text segment instead of wrapping pre-styled output.

### Files changed (4)

1. **`internal/cli/tui/docs/model.go`**
   - `renderTreeItem` now accepts `selected, unfocused bool` params
   - When selected+focused: all segments (arrow, name, count, connector, title) render with `accentColor` + bold
   - When selected+unfocused: all segments render with gray (245)
   - When unselected: original styling preserved
   - `renderExplorer`: passes selection state to `renderTreeItem` instead of wrapping with `selectedStyle.Render()`

2. **`internal/cli/tui/config/model.go`**
   - `renderListItem` now accepts `selected, unfocused bool` params
   - When selected+focused: config name, connector, filename, badge all render in accent+bold
   - When selected+unfocused: all segments render with gray (245)
   - Section headers always render normally (never selected)
   - `renderExplorer`: passes selection state to `renderListItem` instead of wrapping with `selectedStyle.Render()`

3. **`internal/cli/tui/docs/styles.go`** — Removed unused `selectedStyle` and `unfocusedSelectedStyle` vars

4. **`internal/cli/tui/config/styles.go`** — Same cleanup

### Verification
- `make build` — compiles
- `make lint` — 0 issues  
- `make test` — all unit tests pass