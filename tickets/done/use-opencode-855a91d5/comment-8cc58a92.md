---
id: 8cc58a92-2d94-4e90-9129-cb02801ce204
author: claude
type: review_requested
created: 2026-02-14T12:27:46.214029Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

When Cortex spawns OpenCode ticket agents, the system prompt was set via `agent.cortex.prompt` in `OPENCODE_CONFIG_CONTENT`, which **replaces** OpenCode's built-in provider prompt. Claude Code ticket agents use `--append-system-prompt` which preserves the built-in prompt. This change makes OpenCode ticket agents equivalent by using the `instructions` config field (an array of file paths appended to the system prompt) instead of `agent.prompt`.

## Changes

### `internal/core/spawn/opencode_config.go`
- Added `Instructions []string` field to `OpenCodeConfigContent` with `json:"instructions,omitempty"`
- Added `omitempty` to `Prompt` field in `OpenCodeAgentConfig` so empty prompts aren't serialized
- Extended `GenerateOpenCodeConfigContent` signature with `agentType AgentType` and `systemPromptFilePath string` params
- For `AgentTypeTicketAgent`: leaves `Prompt` empty, sets `config.Instructions = []string{systemPromptFilePath}` (when non-empty)
- For all other agent types (architect, meta): keeps current behavior — sets `Prompt = systemPrompt`

### `internal/core/spawn/spawn.go`
- Updated Spawn call site (line ~307) to pass `req.AgentType` and `systemPromptFilePath`
- Updated Resume call site (line ~464) to pass `req.AgentType` and `""` (resume has no prompt files)

### `internal/core/spawn/spawn_test.go`
- Updated 4 existing `TestGenerateOpenCodeConfigContent*` tests to pass new params (using `AgentTypeArchitect` to preserve existing assertions)
- Updated `TestSpawn_OpenCode_ConfigContent` to verify `instructions` file path instead of `agent.Prompt` for ticket agents
- Added `TestGenerateOpenCodeConfigContent_TicketAgent` — verifies ticket agents get `instructions` instead of `agent.prompt`
- Added `TestGenerateOpenCodeConfigContent_TicketAgentNoFile` — verifies empty file path results in no instructions

## Verification
- `make build` — succeeds
- `make test` — all tests pass
- `make lint` — 0 issues