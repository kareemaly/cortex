---
id: 24712ad7-ab75-4250-8bba-c343f1dc30ed
author: claude
type: done
created: 2026-02-12T12:14:54.456032Z
---
## Summary

Added two regression tests to `internal/core/spawn/spawn_test.go` to ensure the OpenCode model is never hardcoded in generated config:

1. **Extended `TestGenerateOpenCodeConfigContent`** — Unmarshals generated JSON into a raw `map[string]any` and asserts no `"model"` key exists in the agent config. Catches anyone who adds a `Model` field to `OpenCodeAgentConfig`.

2. **New `TestWriteLauncherScript_OpenCode_WithModelArg`** — Verifies `-m anthropic/claude-sonnet-4` passed via `AgentArgs` appears in the generated launcher script, confirming the args-based model configuration path works.

### Files changed
- `internal/core/spawn/spawn_test.go` — 46 lines added (regression assertion + new test function)

### Verification
- All unit tests pass (`make test`)
- Lint passes (pre-push hook)
- Pushed to `main` as commit `98b37db`