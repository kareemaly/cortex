---
id: 8a5ee768-d3ea-43bf-a807-c5f79d4e7858
author: claude
type: done
created: 2026-02-02T15:49:44.80658Z
---
## Summary

Implemented two UX improvements for the ticket detail TUI:

### Changes Made

1. **Scroll Position Preservation** (`model.go:207-218`)
   - Body viewport scroll position is now saved before content refresh
   - Position is restored after SSE updates instead of resetting to top
   - Users can now read long ticket descriptions without losing their place

2. **Collapsible Comments Section** (`model.go:997-1012`)
   - Comments collapse to 15% height when body is focused (was 30%)
   - Comments expand to 70% when focused via Tab key (unchanged)
   - Provides more reading space for ticket body by default

### Files Changed
- `internal/cli/tui/ticket/model.go` - Core logic for both features

### Commit
- `5bb2933` feat(tui): add collapsible comments and scroll preservation

### Verification
- All unit tests pass
- Linter passes with no issues