---
id: e61c1d78-3f92-4f52-a52f-d930086a5634
author: claude
type: review_requested
created: 2026-01-30T13:03:07.226024Z
action:
    type: git_diff
    args:
        commit: 99c7539
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Added comprehensive test coverage to verify that `agent_args` from extended base configs are properly inherited when the project doesn't specify them.

## Changes

### `internal/project/config/config_test.go`
- Added args assertion in `TestLoad_WithExtend` to verify `--work-arg` is inherited from base
- Added new test `TestLoad_WithExtend_TicketArgsInherited` that mirrors the real-world scenario (radius project extending claude-code defaults)

### `internal/project/config/merge_test.go`
- Added test case "base args preserved when project has no ticket config" in `TestMergeConfigs_TicketMapMerge`

## Findings

**The merge logic in `merge.go` is already correct.** All tests pass, confirming that:
1. Base config entries (including args) are properly copied when project has no ticket config
2. Args are properly inherited at the role level when project defines a role but not args
3. The `mergeTicketConfig` and `mergeRoleConfig` functions work as expected

The ticket investigation revealed this was a **test coverage gap**, not a bug in the implementation. The extend/merge functionality works correctly.

## Verification
- `make test` - all tests pass
- `make lint` - 0 issues