---
id: e7423166-e4df-4358-bd00-86641d15d639
author: claude
type: ticket_done
created: 2026-01-28T10:31:12.24546Z
---
## Ticket TUI Overhaul — Row-Based Layout with Unified Comment List

### Summary
Replaced the old left/right split panel layout in the ticket detail TUI (`cortex show`) with a two-row vertical layout featuring a unified comment list. The change simplifies the codebase (net -171 lines) while providing a more intuitive viewing experience.

### Files Modified
1. **`internal/cli/tui/ticket/styles.go`** — Added 6 new row/comment styles (`row2FocusedStyle`, `row2Style`, `commentSelectedStyle`, `rowSeparatorStyle`, `attributeHeaderStyle`, `attributeLabelStyle`, `attributeValueStyle`, `dotStyle`). Changed `reviewRequestedColor` from blue (39) to yellow (214). Removed 5 obsolete panel styles.

2. **`internal/cli/tui/ticket/keys.go`** — Added `KeyTab` and `KeyShiftTab` constants. Rewrote `helpText()` signature to accept `focusedRow int` instead of `splitLayout, rightFocused bool`.

3. **`internal/cli/tui/ticket/model.go`** — Major rewrite:
   - Struct: `leftViewport` → `bodyViewport`, `focusedPanel` → `focusedRow`, `sidebarCursor` → `commentCursor`, `modalIsReview`+`modalItemIndex` → `modalCommentIdx`, removed `splitLayout`
   - Added 13 new functions: `rowHeights`, `renderRow1`, `renderAttributes`, `renderCommentList`, `renderCommentLine`, `commentBadge`, `stripMarkdownPreview`, `truncateToWidth`, `commentVisibleRange`, `renderRowSeparator`, `handleCommentListKey`, `updateRowSizes`, `renderBodyContent`
   - Removed 19 obsolete functions
   - Rewrote 9 existing functions: `View`, `Update`, `handleKeyMsg`, `openDetailModal`, `renderDetailModal`, `renderModalHeader`, `renderModalContent`, `handleDetailModalKey`, `hasReviewRequests`

### Key Decisions
- **Row layout always used**: No toggle between single-column and split — the layout adapts based on terminal width (>=100 chars for wide mode with attributes panel, <100 for narrow body-only).
- **Unified comment list**: All comment types in a single list ordered by `ticket.Comments` array, replacing the old separate REVIEWS/COMMENTS sidebar sections. This aligns with the simplified 4-type comment model.
- **Dynamic row sizing**: Focused row gets 70% of available height, unfocused gets 30%. Tab toggles focus.
- **Direct comment indexing**: Modal now stores `modalCommentIdx` as a direct index into `ticket.Comments`, eliminating the need for `reviewComments()` / `nonReviewComments()` split logic.
- **Review badge color**: Changed from blue to yellow (214) to match the plan specification and differentiate from the blue status color used elsewhere.

### Verification
- `make build` — passes
- `make lint` — 0 issues
- `make test` — all unit tests pass

### Commit
`6d5a21a` — `feat: overhaul ticket TUI to row-based layout with unified comment list`

### Follow-up Notes
- Manual testing recommended: `cortex show <ticket-id>` in wide (>=100) and narrow (<100) terminals
- Test embedded mode from kanban board
- Test with 0 comments, 1 comment, and many comments
- Test SSE: add a comment via MCP tool while viewing to verify live updates