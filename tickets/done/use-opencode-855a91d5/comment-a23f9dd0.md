---
id: a23f9dd0-0381-431f-b1b0-3ba92de9f805
author: claude
type: done
created: 2026-02-14T12:28:26.38938Z
---
## Summary

Fixed OpenCode ticket agents to use `instructions` config field instead of `agent.prompt` for system prompts, preserving OpenCode's built-in provider prompt (equivalent to Claude Code's `--append-system-prompt` behavior).

## Changes

### `internal/core/spawn/opencode_config.go`
- Added `Instructions []string` field to `OpenCodeConfigContent`
- Added `omitempty` to `Prompt` field in `OpenCodeAgentConfig`
- Extended `GenerateOpenCodeConfigContent` with `agentType` and `systemPromptFilePath` params
- Ticket agents: use `instructions` (file path); architect/meta: use `agent.prompt` (existing behavior)

### `internal/core/spawn/spawn.go`
- Updated 2 call sites (Spawn and Resume) to pass new parameters

### `internal/core/spawn/spawn_test.go`
- Updated 5 existing tests for new function signature
- Added 2 new tests: `TestGenerateOpenCodeConfigContent_TicketAgent` and `TestGenerateOpenCodeConfigContent_TicketAgentNoFile`

## Verification
- `make build` — passes
- `make test` — all tests pass
- `make lint` — 0 issues
- Pushed to origin/main (commit 0313551)