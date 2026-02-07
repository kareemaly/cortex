---
id: ebcf49aa-0a53-4466-af92-4f56ff00db8d
author: claude
type: done
created: 2026-02-05T11:38:32.542414Z
---
## Summary

Implemented ticket type display in the `cortex show` command, adding visibility for ticket types (debug, research, chore, work) in both TUI and plain-text output modes.

## Changes Made

### Files Modified
1. **internal/cli/tui/ticket/styles.go**
   - Added type badge styles matching kanban: `debugTypeBadgeStyle` (red), `researchTypeBadgeStyle` (blue), `choreTypeBadgeStyle` (gray)
   - Added `typeBadgeStyle()` function to return appropriate style for ticket type

2. **internal/cli/tui/ticket/model.go**
   - Updated `renderHeader()` to include type badge between ID and title
   - Type badge only shown for non-"work" types (consistent with kanban)
   - Format: `<id> [<type>] <title> ... <status>`

3. **cmd/cortex/commands/ticket_show.go**
   - Added Type line after Status in `printTicketPlainText()` for non-TTY output

## Verification
- Build: ✓ passes
- Lint: ✓ 0 issues
- Tests: ✓ all pass
- Pre-push hooks: ✓ passed

## Merge
- Merged `ticket/show-ticket-type-in` into `main`
- Pushed to `origin main`