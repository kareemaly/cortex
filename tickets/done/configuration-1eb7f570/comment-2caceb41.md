---
id: 2caceb41-0468-4079-a422-b2a0518195bd
author: claude
type: done
created: 2026-01-29T08:47:55.109313Z
---
## Summary

Implemented the configuration extension system that allows project configs to inherit from a base config directory via the `extend` attribute. This enables minimal project setup (single `extend` line) with shared defaults from `~/.cortex/defaults/basic`.

## Key Decisions

1. **Path Resolution Strategy**: Support absolute paths, tilde (~) expansion to home directory, and relative paths resolved from project root. This provides flexibility for different use cases (global defaults, shared team configs, relative configs).

2. **Merge Rules**:
   - Scalars: project wins if non-zero
   - Args slices: project replaces entirely (no append) - prevents unpredictable arg combinations
   - Ticket map: merge entries, project wins on conflict - allows adding new ticket types while overriding existing ones

3. **Circular Detection**: Track visited paths during recursive load to detect and error on circular extend references (including self-reference).

4. **PromptResolver Pattern**: File-level override (project has SYSTEM.md → uses it, inherits KICKOFF.md from base). This allows granular customization without copying all prompts.

5. **Install Changes**: `cortex init` now creates `~/.cortex/defaults/basic/` with full defaults (prompts, config) and minimal project config with just `name` + `extend` line. Projects no longer get local prompts/ directory - inherited from base.

## Files Modified/Created

### New Files (6)
- `internal/project/config/path.go` - ResolvePath, ValidateExtendPath
- `internal/project/config/path_test.go` - Path resolution tests
- `internal/project/config/merge.go` - MergeConfigs, mergeRoleConfig, mergeTicketConfig
- `internal/project/config/merge_test.go` - Merge logic tests
- `internal/prompt/resolver.go` - PromptResolver with fallback
- `internal/prompt/resolver_test.go` - Resolver tests

### Modified Files (8)
- `internal/project/config/errors.go` - Added ExtendPathNotFoundError, CircularExtendError
- `internal/project/config/config.go` - Added Extend field, ResolvedExtendPath(), loadWithVisited()
- `internal/project/config/config_test.go` - Extension tests (extend, override, circular, chained)
- `internal/core/spawn/spawn.go` - Added BaseConfigPath to SpawnRequest, use PromptResolver
- `internal/core/spawn/orchestrate.go` - Pass BaseConfigPath from loaded config
- `internal/daemon/api/architect.go` - Pass BaseConfigPath for architect spawns
- `internal/daemon/api/sessions.go` - Use PromptResolver for approve prompt
- `internal/install/install.go` - setupBasicDefaults(), minimal project config

## Verification

- All unit tests pass (47 new tests added)
- Lint passes with 0 issues
- Build succeeds
- Pushed to main branch (commit e7d2d54)

## Follow-up Notes

- Existing projects using the old init format will continue to work (backward compatible - no extend field means defaults applied)
- Users can customize prompts by creating `.cortex/prompts/` in their project to override specific files from base