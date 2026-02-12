---
id: 24273e55-b82a-4060-b59d-badcca4cd655
author: claude
type: done
created: 2026-02-11T10:51:53.151254Z
---
## Summary

Implemented full opencode agent support in cortex init and defaults management.

### Changes

**New files:**
- `internal/install/defaults/opencode/cortex.yaml` — Default config with `agent: opencode` and full CLI args matching claude-code
- `internal/install/defaults/opencode/CONFIG_DOCS.md` — Configuration guide for opencode projects
- `internal/install/defaults/opencode/prompts/` — All 16 prompt files (meta, architect, ticket/{work,debug,research,chore}) identical to claude-code
- `internal/install/install_test.go` — Tests for `DefaultsDirForAgent` mapping

**Modified files:**
- `internal/install/install.go` — Added `Agent` field to `Options`, `DefaultsDirForAgent()` helper, `setupOpenCodeDefaults()`, updated `setupProject()` to use agent-specific defaults directory
- `cmd/cortex/commands/init.go` — Added `--agent`/`-a` flag (defaults to "claude") with validation
- `cmd/cortex/commands/defaults_upgrade.go` — Added "opencode" to `defaultConfigs` slice
- `internal/install/embed_test.go` — Added `TestCopyEmbeddedDefaultsOpenCode`
- `CLAUDE.md` — Updated agent defaults path and init command docs
- `README.md` — Added opencode to agent options and requirements

### Verification
- `make build` — passes
- `make test` — all tests pass
- `make lint` — 0 issues
- Pre-push hooks passed on merge to main
- Merged to main and pushed