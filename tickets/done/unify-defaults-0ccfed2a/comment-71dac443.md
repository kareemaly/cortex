---
id: 71dac443-53fd-4e9c-9f50-53bd1b62cda0
author: claude
type: review_requested
created: 2026-02-14T09:17:39.412958Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Unify Defaults Directory & Decouple Extend from Config Merging

### Changes Summary

**Directory restructuring:**
- Created `internal/install/defaults/main/prompts/` with all prompt files (architect, meta, ticket/{work,debug,research,chore})
- Deleted `internal/install/defaults/claude-code/` and `internal/install/defaults/opencode/` entirely

**`internal/install/embed.go`:**
- Removed `GetConfigDocs()` function (zero external callers)
- Removed unused `fmt` import

**`internal/install/install.go`:**
- Removed `DefaultsDirForAgent()` function
- Removed `setupClaudeCodeDefaults()` and `setupOpenCodeDefaults()`
- Added `setupMainDefaults()` → calls `CopyEmbeddedDefaults("main", ...)`
- Updated `setupGlobal()` to use single `setupMainDefaults()` call
- Rewrote `setupProject()` to generate self-contained `cortex.yaml` via `generateProjectConfig()` — embeds all agent config inline with `extend: ~/.cortex/defaults/main`

**`internal/project/config/config.go`:**
- Replaced `Load()` + `loadWithVisited()` with a non-recursive implementation
- No more `MergeConfigs`, no recursion, no `isBaseConfig` distinction
- `extend` is validated and stored for prompt resolution only

**`internal/project/config/merge.go`:** Deleted entirely

**`internal/project/config/errors.go`:** Removed `CircularExtendError` struct, its `Error()` method, and `IsCircularExtend()` function

**`cmd/cortex/commands/defaults_upgrade.go`:**
- Changed `defaultConfigs` from `["claude-code", "opencode"]` to `["main"]`
- Added legacy directory cleanup after upgrade
- Added warning about updating existing projects' `extend` field
- Added `path/filepath` import

**`cmd/cortex/commands/config_show.go`:**
- Removed `IsCircularExtend` error check
- Updated `Long` description

**`internal/daemon/api/meta.go`:**
- Changed hardcoded `"claude-code"` to `"main"` in `baseConfigPath`

**Test updates:**
- `embed_test.go`: Rewrote all tests for `"main"`, removed `cortex.yaml` assertions, uses prompt files
- `install_test.go`: Deleted (only had `TestDefaultsDirForAgent`)
- `merge_test.go`: Deleted entirely
- `config_test.go`: Removed `TestLoad_WithExtendOverride`, `TestLoad_CircularExtend`, `TestLoad_SelfExtend`, `TestLoad_ChainedExtend`, `TestLoad_WithExtend_TicketArgsInherited`, `setupBaseConfig`. Rewrote `TestLoad_WithExtend` to verify `resolvedExtendPath` is set correctly. Kept all non-merge tests.

### Verification
- `make build` — compiles successfully
- `make test` — all unit tests pass
- `make lint` — 0 issues