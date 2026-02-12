---
id: 9a99b715-faac-4238-903f-e0923340dacc
author: claude
type: review_requested
created: 2026-02-11T10:46:22.835691Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/b11abb2f-5b19-4ee0-b7ef-a503ffd3e4a4
        commit: f864ee6
---
## Summary

Added OpenCode agent support to the spawn launcher. Previously, the `opencode` agent type fell through to `buildClaudeCommand()` which generated incorrect CLI flags. OpenCode has a fundamentally different interface — instead of `--mcp-config`, `--system-prompt`, and `--settings` flags, it uses a single `OPENCODE_CONFIG_CONTENT` environment variable containing a JSON blob.

## Changes

### New file: `internal/core/spawn/opencode_config.go`
- Defines `OpenCodeConfigContent`, `OpenCodeAgentConfig`, and `OpenCodeMCPConfig` types
- `GenerateOpenCodeConfigContent()` transforms a `ClaudeMCPConfig` + system prompt into OpenCode's JSON format:
  - Combines `MCPServerConfig.Command` + `.Args` into a single `command` array
  - Maps `env` → `environment`
  - Sets `type: "local"` for all MCP servers
  - Embeds system prompt in `agent.cortex.prompt`
  - Sets `permission: {"*": "allow"}` and `mode: "bypassPermissions"`

### Modified: `internal/core/spawn/launcher.go`
- Added `"opencode"` case to the agent type switch in `buildLauncherScript`
- Added `buildOpenCodeCommand()` that builds: `opencode --agent cortex --prompt "$(cat kickoff_file)" [agentArgs...]`
- No `--mcp-config`, `--settings`, `--system-prompt`, `--session-id`, or `--resume` flags (all handled via env var)

### Modified: `internal/core/spawn/spawn.go`
- **Spawn()**: Skips settings generation for opencode; generates and injects `OPENCODE_CONFIG_CONTENT` env var with system prompt
- **Resume()**: Skips settings generation for opencode; generates and injects `OPENCODE_CONFIG_CONTENT` with empty system prompt (resume has no prompts)

### Modified: `internal/core/spawn/spawn_test.go`
- `TestGenerateOpenCodeConfigContent` — verifies JSON structure, command array, env mapping, system prompt
- `TestGenerateOpenCodeConfigContent_EmptySystemPrompt` — verifies resume case with empty prompt
- `TestWriteLauncherScript_OpenCode` — verifies generated script uses `opencode --agent cortex`, exports `OPENCODE_CONFIG_CONTENT`, has no Claude-specific flags

## Verification
- `make build` — compiles successfully
- `make test` — all tests pass (including 3 new tests)
- `make lint` — 0 issues