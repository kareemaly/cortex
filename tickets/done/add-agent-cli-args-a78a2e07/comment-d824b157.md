---
id: d824b157-7a41-4f7a-a1a5-a39e2c1255e7
author: claude
type: ticket_done
created: 2026-01-26T18:09:28.914586Z
---
## Summary

Added `agent_args` configuration field to `cortex.yaml` that threads through the spawn pipeline, allowing extra CLI arguments to be appended to the claude command when spawning ticket sessions.

## Changes

### Files Modified

1. **internal/project/config/config.go** — Added `AgentArgs []string` field (yaml tag: `agent_args`) to the `Config` struct.

2. **internal/core/spawn/spawn.go** — Added `AgentArgs []string` field to `SpawnRequest` struct. Passed `req.AgentArgs` into `LauncherParams` when building the launcher.

3. **internal/core/spawn/orchestrate.go** — Set `AgentArgs: projectCfg.AgentArgs` on the `SpawnRequest` built inside `Orchestrate()`, ensuring the config value flows from project config to the spawn request.

4. **internal/core/spawn/launcher.go** — Added `AgentArgs []string` field to `LauncherParams` struct. In `buildLauncherScript`, appended each arg (shell-quoted) to the `parts` slice after all existing flags.

## Key Decisions

- **Shell quoting**: Each agent arg is individually shell-quoted using the existing `shellQuote` helper, preventing injection and handling args with spaces/special characters.
- **Placement**: Args are appended after all built-in flags to ensure they don't interfere with existing functionality.
- **Minimal scope**: Only added what was needed — no validation of arg values, no UI changes.

## Verification

- `make build` — compiles cleanly
- `make test` — all existing tests pass
- `make lint` — 0 issues