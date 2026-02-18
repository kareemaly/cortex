---
id: 48b147d6-9815-457c-bb24-f183d4e08411
author: claude
type: done
created: 2026-02-13T13:37:33.830515Z
---
## Summary

Implemented OpenCode status plugin injection at spawn time to fix OpenCode agent sessions being permanently stuck at `starting` status.

### Problem
OpenCode sessions had no status reporting because hook generation (`GenerateSettingsConfig`) was explicitly skipped for OpenCode in `spawn.go`. The TUI companion pane showed no meaningful status updates.

### Solution
Inject a TypeScript status plugin into OpenCode at spawn time via `OPENCODE_CONFIG_DIR`, writing a plugin that pushes status updates to the existing `POST /agent/status` endpoint.

### Files Changed
- **`internal/core/spawn/opencode_plugin.go`** (new) — `GenerateOpenCodeStatusPlugin()` generates a TS plugin with baked-in daemon URL/ticket ID/project path mapping OpenCode events to Cortex agent statuses. `WriteOpenCodePluginDir()` creates a temp dir with the plugin at `plugin/cortex-status.ts`, made read-only (0555) to skip OpenCode's dependency auto-install.
- **`internal/core/spawn/launcher.go`** — Added `CleanupDirs []string` to `LauncherParams`; updated EXIT trap to `rm -rf` directories alongside existing `rm -f` for files.
- **`internal/core/spawn/spawn.go`** — Wired plugin injection in both `Spawn()` and `Resume()` for OpenCode agents, setting `OPENCODE_CONFIG_DIR` env var and registering temp dir for cleanup.
- **`internal/core/spawn/opencode_plugin_test.go`** (new) — 4 tests: plugin generation, temp dir creation with permissions, OpenCode spawn integration, Claude agent non-interference.

### Verification
- `make build` ✓
- `make lint` ✓ (0 issues)
- `make test` ✓ (all tests pass)
- Pre-push hooks passed
- Pushed to origin/main as commit cb79338