---
id: ee99cb29-f3a6-44c3-8d91-12c9dd7c3615
author: claude
type: review_requested
created: 2026-01-28T13:11:57.735228Z
action:
    type: git_diff
    args:
        commit: 7956fdc
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Implemented visual and interaction improvements to the ticket detail TUI:

### 1. Header Vertical Padding
- Changed `Padding(0, 1)` to `Padding(1, 1)` in `styles.go:39`
- Updated `rowHeights()` in `model.go:947` to account for taller header (now subtracts 4 instead of 3)

### 2. Bracket Shortcuts
- Added `KeyLeftBracket` and `KeyRightBracket` constants in `keys.go:43-44`
- Updated `handleKeyMsg()` in `model.go:332` to handle `[` and `]` for panel toggle
- Updated help text to show `[Tab/[/]]` instead of `[Tab]`

### 3. Comment Row Rendering with 3-Line Markdown Previews
- Added `CommentRowLines = 4` and `CommentRowPadding = 1` constants in `styles.go:7-8`
- Replaced `renderCommentLine()` with `renderCommentRow()` that renders:
  - Line 1: Badge + repo name (if review) + separator + time-ago
  - Lines 2-4: Glamour-rendered markdown content (truncated to 3 lines)
- Added `renderCommentPreview()` function that uses glamour to render markdown and `ansi.Truncate()` to safely truncate ANSI-styled lines
- Added `applyBackgroundToBlock()` for full-row highlight on selected comments
- Updated `commentVisibleRange()` to account for multi-line rows (5 lines per comment including padding)
- Removed unused `dotStyle` and `stripMarkdownPreview()` function

### Files Changed
- `internal/cli/tui/ticket/styles.go` - Header padding, comment row constants, removed dotStyle
- `internal/cli/tui/ticket/keys.go` - Bracket key constants, updated help text
- `internal/cli/tui/ticket/model.go` - New rendering functions, updated key handling

### Verification
- `make build` - Passes
- `make lint` - Passes (0 issues)
- `make test` - All tests pass