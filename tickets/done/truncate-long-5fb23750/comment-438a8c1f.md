---
id: 438a8c1f-da10-447e-8f2d-5f6d54dc7049
author: claude
type: done
created: 2026-02-12T14:41:45.422815Z
---
## Summary

Fixed long titles in the docs and config explorer panes wrapping to the next line and breaking tree-view indentation.

## Changes

- **`internal/cli/tui/docs/model.go`** — Replaced `lipgloss.NewStyle().Width(maxTitle).Render(title)` with `truncateToWidth(title, maxTitle)` in `renderTreeItem()`. Added `truncateToWidth()` helper function.
- **`internal/cli/tui/config/model.go`** — Replaced `lipgloss.NewStyle().Width(maxName).Render(filename)` with `truncateToWidth(filename, maxName)` in `renderListItem()`. Added `truncateToWidth()` helper function.

The `truncateToWidth()` function (copied from existing `ticket/model.go`) truncates rune-by-rune and appends "…" when the string exceeds the available width, preserving tree-view indentation.

## Verification

- `make build` — compiles cleanly
- `make lint` — 0 issues
- `make test` — all tests pass
- Pushed to origin/main (commit 6240137)