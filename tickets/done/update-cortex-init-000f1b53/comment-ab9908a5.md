---
id: ab9908a5-b904-42f3-9e7a-4ee53b4a75b6
author: claude
type: review_requested
created: 2026-02-14T09:36:59.869633Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Agent Detection in `cortex init` + Project Migration

### Changes

**New files:**

1. **`internal/install/agent.go`** — `AgentAvailability` struct with `DetectAgents()`, `OnlyAgent()`, and `AgentCount()` using `exec.LookPath` for `claude` and `opencode`.

2. **`internal/install/migrate.go`** — `MigrationResult` struct, `DetectAgentFromExtend()` (detects agent from legacy extend paths), `MigrateProjectConfig()` (reads legacy config, generates new one preserving worktrees/custom paths), and `MigrateAllProjects()` (iterates global config's registered projects).

3. **`internal/install/agent_test.go`** — Tests for `DetectAgentFromExtend`, `OnlyAgent`, and `AgentCount` (12 test cases).

4. **`internal/install/migrate_test.go`** — Tests for `MigrateProjectConfig` covering legacy claude, legacy opencode, already-migrated, missing config, custom paths, and architect fallback (6 test cases).

**Modified files:**

5. **`internal/install/deps.go`** — Removed `"claude"` from `requiredDeps` (agent availability now handled by detection logic).

6. **`cmd/cortex/commands/init.go`** — Changed `--agent` default from `"claude"` to `""`. When not set and not `--global-only`:
   - 0 agents → error with install instructions
   - 1 agent → auto-select with message
   - 2 agents + TTY → interactive numbered prompt
   - 2 agents + no TTY → error asking for `--agent` flag
   - When explicitly set → validates binary exists via `exec.LookPath`

7. **`cmd/cortex/commands/defaults_upgrade.go`** — Integrated `MigrateAllProjects()`:
   - In dry-run: shows `=== Project Migration ===` preview
   - In apply: migrates configs before legacy directory removal, prints per-project results
   - Removed manual "Note:" message (migration is now automatic)

### Verification

- `make build` — compiles successfully
- `make test` — all tests pass (0 failures), including 18 new test cases