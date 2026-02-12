---
id: 39767d99-d518-4092-88a1-b821ed442702
author: claude
type: done
created: 2026-02-11T14:10:59.714395Z
---
## Summary: Remove GitHub Copilot Agent Integration

Completely removed the GitHub Copilot agent integration since OpenCode now covers the same use case.

### Changes (23 files, +26/-546 lines)

**Deleted:**
- `internal/install/defaults/copilot/` — entire directory (cortex.yaml, CONFIG_DOCS.md, 11 prompt files)

**Go source (7 files):**
- `internal/project/config/config.go` — Removed `AgentCopilot` constant; validation now only accepts `claude` or `opencode`
- `internal/core/spawn/launcher.go` — Removed `buildCopilotCommand()` function and its case branch
- `internal/core/spawn/spawn.go` — Simplified settings skip conditions and removed copilot-only `if` guards around system prompt loading (5 locations total)
- `cmd/cortex/commands/init.go` — Removed copilot from `--agent` flag and validation
- `internal/install/install.go` — Removed copilot from `DefaultsDirForAgent()`, `setupCopilotDefaults()`, and setup call
- `cmd/cortex/commands/defaults_upgrade.go` — Removed copilot from upgrade config list

**Tests (2 files):**
- `internal/install/install_test.go` — Removed copilot test case
- `internal/session/store_test.go` — Changed copilot→opencode in TestList

**Documentation (4 files):**
- CLAUDE.md, README.md — Removed copilot from all references
- Two meta SYSTEM.md prompt files — Updated agent option comments

### Verification
- `make build` ✓
- `make test` ✓ (all tests pass)
- `make lint` ✓ (0 issues)
- Pre-push hooks ✓
- Pushed to origin/main as commit f4df2eb