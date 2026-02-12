---
id: db163aed-d791-4c37-b605-24e4eec5d204
author: claude
type: review_requested
created: 2026-02-11T14:10:13.08881Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary: Remove GitHub Copilot Agent Integration

OpenCode now covers the use case that Copilot served. This removes the entire Copilot agent integration to reduce maintenance burden and dead code.

### Changes

**Deleted:**
- `internal/install/defaults/copilot/` — entire directory (cortex.yaml, CONFIG_DOCS.md, all prompts)

**Modified (Go source):**
- `internal/project/config/config.go` — Removed `AgentCopilot` constant; updated `Validate()` to only accept `claude` or `opencode`
- `internal/core/spawn/launcher.go` — Removed `case "copilot"` branch and entire `buildCopilotCommand()` function; updated comment
- `internal/core/spawn/spawn.go` — Removed copilot from ResumeRequest comment; changed settings skip condition from `!= "copilot" && != "opencode"` to just `!= "opencode"` (2 locations); removed copilot-only `if` wrappers around system prompt loading (3 locations — ticket, architect, meta)
- `cmd/cortex/commands/init.go` — Removed copilot from `--agent` flag description and validation
- `internal/install/install.go` — Removed `case "copilot"` from `DefaultsDirForAgent()`; removed `setupCopilotDefaults()` call and function; updated comment
- `cmd/cortex/commands/defaults_upgrade.go` — Removed `"copilot"` from `defaultConfigs` slice

**Modified (tests):**
- `internal/install/install_test.go` — Removed `{"copilot", "copilot"}` test case
- `internal/session/store_test.go` — Changed `"copilot"` to `"opencode"` in `TestList`

**Modified (docs):**
- `CLAUDE.md` — Removed copilot from agent defaults, config description, and CLI usage
- `README.md` — Removed copilot from requirements, config examples (replaced `copilot` with `opencode` in mixed agents example)
- `internal/install/defaults/claude-code/prompts/meta/SYSTEM.md` — Updated agent comment
- `internal/install/defaults/opencode/prompts/meta/SYSTEM.md` — Updated agent comment

### Verification
- `make build` — compiles without errors
- `make test` — all unit tests pass
- `make lint` — 0 issues
- `grep -ri copilot` in source dirs — zero hits outside `tickets/` and `docs/` (historical records)