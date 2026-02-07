---
id: 91ccee1c-5e10-4146-8108-4fd743e71f56
author: claude
type: done
created: 2026-02-02T11:55:02.933211Z
---
## Summary

Fixed config extend path resolution for base configs. The config loader was incorrectly looking for `.cortex/cortex.yaml` inside extend paths, but installed defaults have `cortex.yaml` directly at the root.

## Changes Made

### `internal/project/config/config.go`
- Added `isBaseConfig` parameter to `loadWithVisited` function
- When `isBaseConfig` is true (loading via extend), looks for `cortex.yaml` at root
- When `isBaseConfig` is false (loading a project), looks for `.cortex/cortex.yaml`
- Renamed `projectRoot` parameter to `configRoot` for clarity

### `internal/project/config/config_test.go`
- Updated `setupBaseConfig` helper to create `cortex.yaml` at root (not in `.cortex` subdirectory)
- Updated `TestLoad_CircularExtend` to use proper base config structure

## Verification
- All unit tests pass
- Linting passes
- `cortex config show` correctly displays merged config with inherited args (`--permission-mode plan`)

## Commit
`62bbb88` - fix(config): resolve extend paths to cortex.yaml directly