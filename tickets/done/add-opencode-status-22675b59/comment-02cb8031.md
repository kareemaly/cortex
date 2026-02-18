---
id: 02cb8031-3dd3-414c-983f-380ca528f248
author: claude
type: review_requested
created: 2026-02-13T13:37:01.339996Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
        commit: cb79338
---
## Summary

Implemented OpenCode status plugin injection at spawn time to fix OpenCode sessions being permanently stuck at `starting` status.

### Changes

**New file: `internal/core/spawn/opencode_plugin.go`**
- `GenerateOpenCodeStatusPlugin()` — generates a TypeScript plugin with baked-in daemon URL, ticket ID, and project path. Maps OpenCode events to Cortex agent statuses (`session.status` → busy/idle/retry, `permission.asked` → waiting_permission, `tool.execute.before/after` → in_progress). Uses fire-and-forget `fetch()` with `AbortSignal.timeout(5000)`.
- `WriteOpenCodePluginDir()` — creates a temp dir with `plugin/cortex-status.ts`, makes the plugin dir read-only (0555) to skip OpenCode's dependency auto-install.

**Modified: `internal/core/spawn/launcher.go`**
- Added `CleanupDirs []string` field to `LauncherParams`
- Updated `buildLauncherScript()` trap to include `rm -rf` for directories alongside existing `rm -f` for files

**Modified: `internal/core/spawn/spawn.go`**
- Added plugin injection in both `Spawn()` and `Resume()` paths for OpenCode agents
- Sets `OPENCODE_CONFIG_DIR` env var pointing to the temp plugin directory
- Plugin dir is registered in `CleanupDirs` for automatic cleanup on session exit
- Uses `daemonconfig.DefaultDaemonURL` for the daemon URL and reads `CORTEX_TICKET_ID` from the already-set env vars (handles both ticket and architect sessions)

**New file: `internal/core/spawn/opencode_plugin_test.go`**
- `TestGenerateOpenCodeStatusPlugin` — verifies baked-in values, event handlers, status mappings, timeout, and headers
- `TestWriteOpenCodePluginDir` — verifies temp dir creation, file placement, and read-only permissions
- `TestOpenCodeSpawnIncludesPluginDir` — integration test: spawns OpenCode agent, verifies `OPENCODE_CONFIG_DIR` export and `rm -rf` in trap
- `TestClaudeSpawnDoesNotIncludePluginDir` — verifies Claude agent is not affected

### Verification
- `make build` ✓
- `make lint` ✓ (0 issues)
- `make test` ✓ (all tests pass including 4 new tests)