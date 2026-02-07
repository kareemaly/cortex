---
id: 97d6f706-608d-47c9-8e34-ea9fc4c8beee
author: claude
type: done
created: 2026-01-28T13:16:23.44477Z
---
## Ticket TUI Improvements - Complete Report

### Summary
Implemented visual and interaction improvements to the ticket detail TUI as specified in the ticket requirements. All acceptance criteria have been met.

### Changes Made

#### 1. Header Vertical Padding
- **File:** `internal/cli/tui/ticket/styles.go:39`
- Changed `Padding(0, 1)` to `Padding(1, 1)` for top and bottom padding
- **File:** `internal/cli/tui/ticket/model.go:947`
- Updated `rowHeights()` to subtract 4 instead of 3 to account for the taller header (header now takes 3 lines instead of 1)

#### 2. Bracket Shortcuts for Panel Switching
- **File:** `internal/cli/tui/ticket/keys.go:43-44`
- Added `KeyLeftBracket = "["` and `KeyRightBracket = "]"` constants
- **File:** `internal/cli/tui/ticket/model.go:332`
- Updated key handler to accept `[` and `]` for panel focus toggle (same behavior as Tab/Shift+Tab)
- **File:** `internal/cli/tui/ticket/keys.go:68-70`
- Updated help text from `[Tab]` to `[Tab/[/]]` for both body and comments panels

#### 3. Comment Row Padding and Full-Row Highlight with 3-Line Markdown Previews
- **File:** `internal/cli/tui/ticket/styles.go:7-8`
- Added constants: `CommentRowLines = 4` (header + 3 preview lines) and `CommentRowPadding = 1` (blank line between rows)
- **File:** `internal/cli/tui/ticket/model.go`
  - Replaced `renderCommentLine()` with `renderCommentRow()` - renders 4-line comment rows:
    - Line 1: Badge + repo name (for reviews) + separator dashes + time-ago
    - Lines 2-4: Glamour-rendered markdown content
  - Added `renderCommentPreview()` - renders markdown with glamour and truncates to maxLines using `ansi.Truncate()` for ANSI-safe truncation
  - Added `plainTextPreview()` - fallback for plain text when glamour fails
  - Added `applyBackgroundToBlock()` - applies background color to all lines of a selected comment row
  - Updated `commentVisibleRange()` to calculate visible count based on row height (5 lines per comment including padding)
  - Updated `renderCommentList()` to call new `renderCommentRow()` and add padding between rows
- Removed unused `dotStyle` from styles.go
- Removed unused `stripMarkdownPreview()` function from model.go
- Added import for `github.com/charmbracelet/x/ansi` for ANSI-safe truncation

### Key Decisions

1. **Truncation strategy**: Render markdown first with glamour, then truncate the rendered ANSI output (not raw markdown). This prevents broken markdown syntax and garbled output.

2. **Row height calculation**: Each comment takes 5 lines total (4 content + 1 padding), simplifying the visible range calculation.

3. **Full-row highlight**: Applied background to each line individually after padding to full width, ensuring consistent highlight across the entire row.

4. **Separator in header**: Used dashes (─) between badge/repo and time-ago for visual separation within comment rows.

### Files Modified
- `internal/cli/tui/ticket/styles.go` - Header padding, comment row constants, removed dotStyle
- `internal/cli/tui/ticket/keys.go` - Bracket key constants, updated help text
- `internal/cli/tui/ticket/model.go` - New rendering functions, updated key handling, updated height calculations

### Verification
- `make build` - Passes
- `make lint` - 0 issues
- `make test` - All tests pass

### Commit
- `7956fdc` - feat: improve ticket TUI with header padding, bracket shortcuts, and 3-line comment previews