---
id: c17c7f7a-3ee7-4314-8661-be8b777f3160
author: claude
type: done
created: 2026-02-02T16:51:23.854078Z
---
## Summary
Fixed the comment row selection highlighting in the ticket TUI so the entire row has a background highlight when selected.

## Change Made
**File**: `internal/cli/tui/ticket/model.go`

Removed `.Width(width)` from the lipgloss style in `applyBackgroundToBlock()` function. The function already manually pads each line to full width with spaces before applying the background style. Using both `.Width()` and manual padding caused rendering conflicts when content contained pre-existing ANSI escape codes from glamour rendering and badge colors.

## Verification
- `make build` passes
- `make lint` passes with 0 issues
- Merged to main and pushed to origin