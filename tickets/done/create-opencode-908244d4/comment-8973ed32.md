---
id: 8973ed32-30c8-480b-935c-c31afdf7b321
author: claude
type: review_requested
created: 2026-02-11T10:41:23.110586Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/168c656b-188f-49d6-8360-09bb255f3f91
        commit: e88d50d
---
## Summary

Added `internal/install/defaults/opencode/` directory with 16 files so that `cortex init` with OpenCode produces a working setup out of the box.

### Files created:

1. **`cortex.yaml`** — Uses `agent: opencode` for all roles (architect, meta, ticket/*) with empty args. OpenCode handles permissions via `OPENCODE_CONFIG_CONTENT` env var rather than CLI flags like `--allowedTools`.

2. **`CONFIG_DOCS.md`** — OpenCode-specific configuration guide documenting the empty-args approach and how to customize prompts.

3. **14 prompt files** (copied verbatim from `claude-code`):
   - `prompts/architect/{SYSTEM,KICKOFF}.md`
   - `prompts/meta/{SYSTEM,KICKOFF}.md`
   - `prompts/ticket/{work,debug,research,chore}/{SYSTEM,KICKOFF,APPROVE}.md`

### No code changes required:
- `internal/install/embed.go` uses `//go:embed defaults/*` which auto-includes the new subdirectory
- `internal/project/config/config.go` already has `AgentOpenCode` constant
- `internal/core/spawn/launcher.go` already handles opencode

### Verification:
- `make build` — succeeds, files are properly embedded
- `make test` — all tests pass