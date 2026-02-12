---
id: 162ce897-deaa-44f1-9da3-e0fa4b4deaa6
author: claude
type: done
created: 2026-02-11T13:16:30.771186Z
---
## Summary

Added 9 integration tests and 2 helpers to `internal/core/spawn/spawn_test.go` covering the OpenCode agent spawn lifecycle through the Orchestrate layer.

### Changes
- **File**: `internal/core/spawn/spawn_test.go` (+574 lines)

### New Helpers
1. `extractExportedEnvVar` — parses shell-quoted env vars from launcher scripts (reverses shellQuote escaping)
2. `createTestCortexConfig` — creates `.cortex/cortex.yaml` in test directories

### New Tests
| Test | Coverage |
|------|----------|
| `TestSpawn_OpenCode_Success` | Spawn succeeds, correct command, no Claude flags, session records "opencode" |
| `TestSpawn_OpenCode_ConfigContent` | Full JSON verification of OPENCODE_CONFIG_CONTENT (agent, MCP, environment) |
| `TestOrchestrate_OpenCode_Normal` | Full Orchestrate: OutcomeSpawned, backlog→progress move, opencode launcher |
| `TestOrchestrate_OpenCode_Resume_Orphaned` | Resume: OutcomeResumed, empty prompt, no --resume flag |
| `TestOrchestrate_OpenCode_AgentFromConfig` | Agent resolved from cortex.yaml without explicit Agent field |
| `TestGenerateOpenCodeConfigContent_MultipleServers` | Two MCP servers transformed correctly |
| `TestGenerateOpenCodeConfigContent_SpecialCharsInPrompt` | Special chars survive JSON round-trip |
| `TestWriteLauncherScript_OpenCode_NoPrompt` | No --prompt flag when PromptFilePath empty |
| `TestWriteLauncherScript_OpenCode_Resume` | No --resume flag (unsupported by design) |

### Verification
- All 49 spawn tests pass
- Full `make test` passes
- `make lint` passes (fixed gofmt issue in comment)
- Merged to main and pushed