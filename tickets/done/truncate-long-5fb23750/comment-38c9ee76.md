---
id: 38c9ee76-0934-4a92-9626-a648268cbf2f
author: claude
type: review_requested
created: 2026-02-12T14:41:06.231261Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
        commit: "6240137"
---
## Summary

Fixed long titles in the docs and config explorer panes wrapping to the next line and breaking tree-view indentation.

## Changes

- **`internal/cli/tui/docs/model.go`** — In `renderTreeItem()`, replaced `lipgloss.NewStyle().Width(maxTitle).Render(title)` (which wraps) with `truncateToWidth(title, maxTitle)` (which truncates with `…`). Added `truncateToWidth()` helper function.

- **`internal/cli/tui/config/model.go`** — In `renderListItem()`, replaced `lipgloss.NewStyle().Width(maxName).Render(filename)` with `truncateToWidth(filename, maxName)`. Added same `truncateToWidth()` helper function.

The `truncateToWidth()` function was copied from the existing implementation in `internal/cli/tui/ticket/model.go` — it truncates rune-by-rune and appends `…` when the string exceeds `maxWidth`.

## Verification

- `make build` — compiles cleanly
- `make lint` — 0 issues
- `make test` — all tests pass