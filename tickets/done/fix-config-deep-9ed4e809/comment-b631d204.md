---
id: b631d204-1e8f-4fb8-ad95-9b1e7eacea9d
author: claude
type: done
created: 2026-01-30T13:09:52.356989Z
---
## Summary

Investigated and added test coverage for config deep merge with extended bases. The merge logic was already correct - this ticket revealed a test coverage gap rather than a bug.

## Changes Made

### `internal/project/config/config_test.go`
- Added args assertion in `TestLoad_WithExtend` (line 458-460) to verify `--work-arg` is inherited from base config
- Added new integration test `TestLoad_WithExtend_TicketArgsInherited` that mirrors the real-world scenario of a project (like radius) extending claude-code defaults without defining its own ticket config

### `internal/project/config/merge_test.go`
- Added test case "base args preserved when project has no ticket config" in `TestMergeConfigs_TicketMapMerge` to verify args inheritance at the unit level

## Key Findings

1. **The merge logic in `merge.go` is correct** - `mergeTicketConfig` properly copies base entries when project has no ticket config
2. **Args inheritance works at the role level** - `mergeRoleConfig` correctly uses base args when project role has none
3. **The issue was test coverage** - existing tests verified agent inheritance but not args inheritance

## Verification

- `make test` - all tests pass
- `make lint` - 0 issues
- Commit: `99c7539` pushed to main

## Follow-up Notes

If args still aren't being applied in practice (e.g., radius project), the issue may be in how the spawn logic consumes the config, not in how configs are loaded/merged. The config loading tests now confirm the merge is correct.