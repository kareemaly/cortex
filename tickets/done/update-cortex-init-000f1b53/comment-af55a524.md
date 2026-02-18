---
id: af55a524-7efd-4e66-ba41-15909d1f2d16
author: claude
type: done
created: 2026-02-14T09:38:08.687478Z
---
## Summary

Implemented agent auto-detection in `cortex init` and automated project config migration in `cortex defaults upgrade`.

### Changes (7 files, commit 1a29a93)

**New files:**
- `internal/install/agent.go` — `DetectAgents()` checks PATH for `claude`/`opencode`, with `OnlyAgent()` and `AgentCount()` helpers
- `internal/install/migrate.go` — `MigrateProjectConfig()` rewrites legacy configs (extend pointing to `defaults/claude-code` or `defaults/opencode`) to use `defaults/main`, preserving worktrees and custom docs/tickets paths; `MigrateAllProjects()` iterates all registered projects
- `internal/install/agent_test.go` — 12 test cases for detection helpers
- `internal/install/migrate_test.go` — 6 test cases covering legacy claude/opencode, already-migrated, missing config, custom paths, architect fallback

**Modified files:**
- `internal/install/deps.go` — Removed `"claude"` from `requiredDeps` (agent availability handled by new detection logic)
- `cmd/cortex/commands/init.go` — `--agent` default changed from `"claude"` to `""` with auto-detection: 0 agents = error with install links, 1 = auto-select, 2 + TTY = interactive prompt, 2 + no TTY = error. Explicit `--agent` validates binary exists via `exec.LookPath`
- `cmd/cortex/commands/defaults_upgrade.go` — Integrated `MigrateAllProjects()` in both dry-run (preview) and apply (migrate before legacy dir removal) paths; removed manual "Note:" message

### Verification
- `make build` — passes
- `make test` — all tests pass (18 new test cases, 0 failures)
- `make lint` — 0 issues (verified by pre-push hook)
- Pushed to origin/main