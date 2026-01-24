# Kanban Column Margins

## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

Large white space appears next to the last column in kanban view. Left and right margins are uneven.

## Requirements

- Left and right margins should be equal and compact
- Columns should fill available width evenly

## Implementation

### Commits
- `fdbf8dc` fix: center kanban columns to distribute leftover width equally
- `d311012` fix: make kanban column margins more compact

### Key Files Changed
- `internal/cli/tui/kanban/model.go` - Adjusted width calculation and removed centering

### Approach
Initial fix (fdbf8dc) centered columns but margins were still too large. Final fix:
1. Changed width calculation from `(m.width-8)/4` to `(m.width-2)/4` - columns use more available width
2. Removed `PlaceHorizontal` centering - no longer needed since columns fill the width

Result: Compact margins (~1 char each side) with columns filling available space.
