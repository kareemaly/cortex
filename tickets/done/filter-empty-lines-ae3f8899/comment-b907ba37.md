---
id: b907ba37-f88e-4303-be8a-3b0073699cdc
author: claude
type: review_requested
created: 2026-02-02T16:23:52.288813Z
action:
    type: git_diff
    args:
        commit: 10c60cd
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/1855c31c-23bd-46cc-bf50-8aed074a740c
---
## Summary

Changed `renderCommentPreview()` in `internal/cli/tui/ticket/model.go` to filter **all** empty/whitespace-only lines, not just those at the start of the preview.

## Change

**Before** (line 1241-1243):
```go
// Skip empty lines at the start.
if len(result) == 0 && strings.TrimSpace(line) == "" {
    continue
}
```

**After**:
```go
// Skip empty lines.
if strings.TrimSpace(line) == "" {
    continue
}
```

## Impact

Comment previews now show 3 lines of actual content instead of potentially including blank lines in the middle. This makes better use of the limited preview space.

## Verification

- `make build` - compiles successfully
- `make test` - all unit tests pass