---
id: b4e723e9-9dc2-4268-8f3b-e71b3af5d287
author: claude
type: review_requested
created: 2026-01-29T08:47:12.286748Z
action:
    type: git_diff
    args:
        commit: e7d2d54
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Implemented the configuration extension system that allows project configs to inherit from a base config via the `extend` attribute. This enables minimal project setup (just name + extend line) with shared defaults.

## Changes

### New Files
- `internal/project/config/path.go` - Path resolution utilities supporting absolute, tilde (~), and relative paths
- `internal/project/config/merge.go` - Deep merge logic for configs (scalars: project wins, args: project replaces, ticket map: merge entries)
- `internal/prompt/resolver.go` - PromptResolver with project-first, base-fallback loading

### Modified Files
- `internal/project/config/errors.go` - Added ExtendPathNotFoundError and CircularExtendError
- `internal/project/config/config.go` - Added Extend field, modified Load() with recursive extension and circular reference detection
- `internal/core/spawn/spawn.go` - Added BaseConfigPath to SpawnRequest, updated prompt loading to use PromptResolver
- `internal/core/spawn/orchestrate.go` - Pass BaseConfigPath from loaded config
- `internal/daemon/api/architect.go` - Pass BaseConfigPath for architect spawns
- `internal/daemon/api/sessions.go` - Use PromptResolver for approve prompt loading
- `internal/install/install.go` - Create ~/.cortex/defaults/basic/ with full defaults, minimal project config with extend

### Tests
- `path_test.go` - Path resolution tests (absolute, tilde, relative, validation)
- `merge_test.go` - Config merge tests (scalars, args, ticket map, git config)
- Extension tests in `config_test.go` (extend, override, circular detection, chained extend)
- `resolver_test.go` - PromptResolver tests (project-first, base-fallback, partial override)

## Verification
- All unit tests pass (`make test`)
- Lint passes (`make lint`)
- Build succeeds (`make build`)