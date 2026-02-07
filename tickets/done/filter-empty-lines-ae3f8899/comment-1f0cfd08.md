---
id: 1f0cfd08-fda4-4197-8090-9091be1047ab
author: claude
type: done
created: 2026-02-02T16:28:52.83939Z
---
## Summary

Fixed comment preview to filter all empty lines, not just those at the start.

## Changes

**File modified:** `internal/cli/tui/ticket/model.go`

Changed `renderCommentPreview()` condition from:
```go
if len(result) == 0 && strings.TrimSpace(line) == "" {
```
to:
```go
if strings.TrimSpace(line) == "" {
```

## Impact

Comment previews now display 3 lines of actual content instead of potentially including blank lines in the middle, making better use of limited preview space.

## Verification

- Build passes (`make build`)
- All unit tests pass (`make test`)
- Merged to main and pushed to origin