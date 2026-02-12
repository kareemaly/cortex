---
id: c0504e02-5801-40a1-8400-1073cc43672d
author: claude
type: review_requested
created: 2026-02-11T10:48:37.086996Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/3bddd732-fb22-4233-b640-1c498484feb3
        commit: aa160b5
---
## Summary

Completed opencode support in cortex init and defaults management. Changes:

### New files
- **`internal/install/defaults/opencode/`** — Full defaults directory mirroring claude-code structure:
  - `cortex.yaml` — Same as claude-code but with `agent: opencode` throughout
  - `CONFIG_DOCS.md` — Configuration docs referencing opencode and `~/.cortex/defaults/opencode`
  - `prompts/` — All 16 prompt files (meta, architect, ticket/{work,debug,research,chore}) identical to claude-code
- **`internal/install/install_test.go`** — Tests for `DefaultsDirForAgent` mapping function

### Modified files
- **`internal/install/install.go`**:
  - Added `Agent` field to `Options` struct
  - Added `DefaultsDirForAgent()` helper mapping claude→claude-code, opencode→opencode, copilot→copilot
  - Added `setupOpenCodeDefaults()` called from `setupGlobal()`
  - Updated `setupProject()` to use `DefaultsDirForAgent(agent)` instead of hardcoded "claude-code"
- **`cmd/cortex/commands/init.go`**:
  - Added `--agent`/`-a` flag defaulting to "claude"
  - Added validation (claude, opencode, copilot)
  - Passes agent to `install.Options`
- **`cmd/cortex/commands/defaults_upgrade.go`**:
  - Added "opencode" to `defaultConfigs` slice
- **`internal/install/embed_test.go`**:
  - Added `TestCopyEmbeddedDefaultsOpenCode` verifying all 18 expected files are extracted and cortex.yaml contains "agent: opencode"

### Verification
- `make build` — passes
- `make test` — all tests pass (including new ones)
- `make lint` — 0 issues
- No changes needed for eject (already generic) or embed.go (auto-picks up new directory)