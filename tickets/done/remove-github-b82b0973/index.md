---
id: b82b0973-7aa9-490e-aaa9-49d32e2dd4ee
title: Remove GitHub Copilot agent integration entirely
type: work
tags:
    - cleanup
    - opencode
created: 2026-02-11T14:04:41.813008Z
updated: 2026-02-11T14:10:59.715864Z
---
## Objective

Completely remove all GitHub Copilot agent support from the codebase. OpenCode now handles this use case, so the Copilot integration is no longer needed.

## What to remove

Remove **everything** related to the Copilot agent type. Leave nothing behind. This includes but is not limited to:

- **Defaults**: `internal/install/defaults/copilot/` — entire directory (cortex.yaml, prompts, CONFIG_DOCS.md)
- **Config**: `AgentCopilot` constant and any copilot references in `internal/project/config/config.go`
- **Launcher**: `buildCopilotCommand()` and the `"copilot"` case in the agent type switch in `internal/core/spawn/launcher.go`
- **Spawn logic**: Any copilot-specific branching in `internal/core/spawn/spawn.go` (e.g., skipping settings for copilot)
- **Init flow**: Remove copilot as an agent choice from `cortex init` in `cmd/cortex/commands/init.go`
- **Install logic**: Remove copilot handling from `internal/install/install.go` (DefaultsDirForAgent, setupCopilotDefaults or similar)
- **Defaults upgrade**: Remove "copilot" from `defaultConfigs` in `cmd/cortex/commands/defaults_upgrade.go`
- **Tests**: Remove all copilot-specific tests (spawn tests, install tests, embed tests)
- **Documentation**: Remove copilot references from CLAUDE.md, README.md, any CONFIG_DOCS, and prompt files that mention copilot
- **MCP tools**: Check if any MCP tool descriptions or prompts reference copilot
- **Any other references**: Search the entire codebase for "copilot", "Copilot", "gh copilot", "AgentCopilot" and remove all occurrences

## Acceptance criteria

- Zero references to "copilot" remain in the codebase (except possibly git history)
- `AgentCopilot` constant is gone
- `internal/install/defaults/copilot/` directory is deleted
- `buildCopilotCommand()` is gone
- `cortex init` only offers claude and opencode
- `make build`, `make test`, and `make lint` all pass
- No dead code or orphaned config paths left behind