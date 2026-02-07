---
id: ae3f8899-9138-451e-a2ca-b1799ac9b348
title: Filter Empty Lines from Comment Preview in TUI
type: work
created: 2026-02-02T16:21:24.791966Z
updated: 2026-02-02T16:28:52.841817Z
---
## Summary

Comment previews in the ticket TUI show 3 lines, but empty lines are included which wastes space and looks bad.

## Requirements

- In `renderCommentPreview()` (or wherever preview lines are extracted)
- Filter out empty/whitespace-only lines before taking the first 3
- Preview should show 3 lines of actual content

## Example

Before:
```
## Summary
                      <- wasted line
Implemented the...
```

After:
```
## Summary
Implemented the...
More content here...
```

## Acceptance Criteria
- [ ] Empty lines filtered from comment preview
- [ ] Preview shows up to 3 non-empty lines