---
id: f291dd3b-514c-446d-9980-0066212042b7
author: claude
type: done
created: 2026-02-11T10:47:27.797502Z
---
## Summary

Added `buildOpenCodeCommand` to the spawn launcher so the `opencode` agent type generates correct CLI commands instead of falling through to Claude's command builder.

## Changes

### New file: `internal/core/spawn/opencode_config.go`
- `OpenCodeConfigContent`, `OpenCodeAgentConfig`, `OpenCodeMCPConfig` types for OpenCode's JSON config format
- `GenerateOpenCodeConfigContent()` transforms `ClaudeMCPConfig` + system prompt into OpenCode's `OPENCODE_CONFIG_CONTENT` JSON — combines command+args into a single array, maps env→environment, sets type "local", embeds system prompt in agent config

### Modified: `internal/core/spawn/launcher.go`
- Added `"opencode"` case to the agent type switch in `buildLauncherScript`
- `buildOpenCodeCommand()` generates: `opencode --agent cortex --prompt "$(cat file)" [agentArgs...]`

### Modified: `internal/core/spawn/spawn.go`
- `Spawn()`: Skips settings generation for opencode; generates and injects `OPENCODE_CONFIG_CONTENT` env var with system prompt
- `Resume()`: Skips settings generation for opencode; generates and injects `OPENCODE_CONFIG_CONTENT` with empty system prompt

### Modified: `internal/core/spawn/spawn_test.go`
- 3 new tests: `TestGenerateOpenCodeConfigContent`, `TestGenerateOpenCodeConfigContent_EmptySystemPrompt`, `TestWriteLauncherScript_OpenCode`

## Verification
- `make build` — clean
- `make test` — all pass
- `make lint` — 0 issues
- Pre-push hooks passed on `git push origin main`