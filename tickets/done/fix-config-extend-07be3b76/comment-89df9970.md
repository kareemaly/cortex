---
id: 89df9970-e4b0-41b5-9642-392b7710fc96
author: claude
type: review_requested
created: 2026-02-02T11:54:09.239355Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Fixed config extend path resolution so that base configs are loaded correctly. The issue was that the config loader expected `.cortex/cortex.yaml` inside the extend path, but installed defaults have `cortex.yaml` directly at the root.

## Changes

### `internal/project/config/config.go`
- Modified `loadWithVisited` to accept a new `isBaseConfig` parameter that controls whether to look for `cortex.yaml` at root (base configs) or `.cortex/cortex.yaml` (project configs)
- When loading base configs via extend, now looks for `cortex.yaml` directly instead of `.cortex/cortex.yaml`

### `internal/project/config/config_test.go`
- Updated `setupBaseConfig` helper to create `cortex.yaml` at root (not in `.cortex` subdirectory)
- Updated `TestLoad_CircularExtend` to use proper base config structure for circular reference detection

## Verification
- All unit tests pass (`make test`)
- Linting passes (`make lint`)
- Verified `cortex config show` displays correctly merged config with inherited args (`--permission-mode plan`)

## Acceptance Criteria
- [x] Extend paths resolve `cortex.yaml` directly (not `.cortex/cortex.yaml`)
- [x] Existing tests updated to reflect new behavior
- [x] `cortex config show` displays correctly merged config when extending defaults
- [x] Plan mode args are inherited when extending `~/.cortex/defaults/claude-code`