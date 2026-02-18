---
id: 0fa0a1b9-6736-4847-86a9-6088a5547d5e
author: claude
type: review_requested
created: 2026-02-12T12:12:26.684189Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
        commit: 98b37db
---
## Changes

Added two regression tests in `internal/core/spawn/spawn_test.go` to ensure the OpenCode model is never hardcoded:

1. **No `model` key in generated JSON** — Extended `TestGenerateOpenCodeConfigContent` to unmarshal the raw JSON into `map[string]any` and assert the `"cortex"` agent config does not contain a `"model"` key. If someone adds a `Model` field to `OpenCodeAgentConfig`, this test will catch it.

2. **Model arg flows through to launcher script** — New `TestWriteLauncherScript_OpenCode_WithModelArg` verifies that `AgentArgs: []string{"-m", "anthropic/claude-sonnet-4"}` produces a launcher script containing `'-m' 'anthropic/claude-sonnet-4'`, confirming the args-based configuration path works.

All tests pass (`make test`).