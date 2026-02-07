---
id: 1080e4d4-f071-4201-9cad-56f3d39f422f
author: claude
type: comment
created: 2026-01-29T08:46:32.578152Z
---
Implementation of configuration extension system completed:

**New files created:**
- `internal/project/config/path.go` - Path resolution utilities (absolute, tilde, relative)
- `internal/project/config/merge.go` - Deep merge logic for configs
- `internal/prompt/resolver.go` - PromptResolver with project/base fallback

**Modified files:**
- `internal/project/config/errors.go` - Added ExtendPathNotFoundError, CircularExtendError
- `internal/project/config/config.go` - Added Extend field, modified Load() with recursive extension support
- `internal/core/spawn/spawn.go` - Added BaseConfigPath to SpawnRequest, updated prompt loading to use PromptResolver
- `internal/core/spawn/orchestrate.go` - Pass BaseConfigPath from loaded config
- `internal/daemon/api/architect.go` - Pass BaseConfigPath for architect spawns
- `internal/daemon/api/sessions.go` - Use PromptResolver for approve prompt loading
- `internal/install/install.go` - Create ~/.cortex/defaults/basic/ with full defaults, minimal project config with extend

**New tests:**
- `internal/project/config/path_test.go` - Path resolution tests
- `internal/project/config/merge_test.go` - Config merge tests
- Extension tests in `internal/project/config/config_test.go`
- `internal/prompt/resolver_test.go` - PromptResolver tests

All tests pass, no lint issues.