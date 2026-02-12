---
id: 66371a3e-f005-4e37-ba9e-35b8e7d4e1fd
author: claude
type: review_requested
created: 2026-02-11T13:14:45.701952Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/c554624c-4481-4f42-beb0-bbf72b9b366c
        commit: 9a6c351
---
## Summary

Added 9 new integration tests and 2 helper functions to `internal/core/spawn/spawn_test.go` covering the OpenCode agent spawn lifecycle through the Orchestrate layer.

### New Helpers

1. **`extractExportedEnvVar`** — Extracts a shell-quoted env var value from a launcher script, reversing `shellQuote` escaping (`'\''` → `'`). Used to parse the `OPENCODE_CONFIG_CONTENT` JSON from launcher export lines.

2. **`createTestCortexConfig`** — Creates `.cortex/cortex.yaml` in test directories, following the existing `createTestPromptFile` pattern. Used by the config-driven agent resolution test.

### New Tests

| Test | What it verifies |
|------|-----------------|
| `TestSpawn_OpenCode_Success` | Spawn with `Agent: "opencode"` succeeds, tmux spawned once, launcher has `opencode --agent cortex`, exports `OPENCODE_CONFIG_CONTENT`, no Claude flags, session records "opencode" |
| `TestSpawn_OpenCode_ConfigContent` | Full JSON parse of `OPENCODE_CONFIG_CONTENT` from launcher: agent mode, permissions, system prompt, MCP type/command/environment |
| `TestOrchestrate_OpenCode_Normal` | Full Orchestrate flow: OutcomeSpawned, session records "opencode", ticket moves backlog→progress, launcher uses OpenCode (not Claude) |
| `TestOrchestrate_OpenCode_Resume_Orphaned` | Resume orphaned session: OutcomeResumed, empty system prompt in config, no `--resume` flag, no Claude flags |
| `TestOrchestrate_OpenCode_AgentFromConfig` | `.cortex/cortex.yaml` with `agent: opencode` → agent resolved from config without explicit Agent field |
| `TestGenerateOpenCodeConfigContent_MultipleServers` | Two MCP servers (cortex + memory) both transformed with correct `type`, `command`, `environment` |
| `TestGenerateOpenCodeConfigContent_SpecialCharsInPrompt` | System prompt with quotes, newlines, backslashes, unicode survives JSON round-trip |
| `TestWriteLauncherScript_OpenCode_NoPrompt` | Empty `PromptFilePath` → no `--prompt` flag, `opencode --agent cortex` present |
| `TestWriteLauncherScript_OpenCode_Resume` | `Resume: true` → no `--resume` in script (OpenCode ignores resume by design) |

### Verification

- All 49 spawn tests pass (`go test ./internal/core/spawn/ -v`)
- Full `make test` passes with no regressions