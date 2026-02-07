---
id: 1c6e6841-345b-486f-86c4-fbab2cefcdbd
author: claude
type: review_requested
created: 2026-02-05T10:20:36.438268Z
action:
    type: git_diff
    args:
        commit: 327a7c9
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Implemented GitHub Copilot CLI as a supported agent type alongside Claude, enabling multi-model flexibility in Cortex.

## Changes Made

### Core Implementation

1. **Agent Type Configuration** (`internal/project/config/config.go`)
   - Added `AgentCopilot AgentType = "copilot"` constant
   - Updated `Validate()` to accept `copilot` as a valid agent type

2. **Agent-Aware Launcher** (`internal/core/spawn/launcher.go`)
   - Added `AgentType` field to `LauncherParams`
   - Created `buildCopilotCommand()` with Copilot-specific flags:
     - `--yolo` (required for automation)
     - `--no-custom-instructions` (ignore AGENTS.md)
     - `--additional-mcp-config` (instead of `--mcp-config`)
   - Refactored `buildLauncherScript()` to dispatch based on agent type

3. **Spawn Flow Updates** (`internal/core/spawn/spawn.go`)
   - Added `Agent` field to `ResumeRequest`
   - Pass agent type to `LauncherParams`
   - Skip SYSTEM.md loading for Copilot (doesn't support `--system-prompt`)
   - Skip settings config for Copilot (doesn't support `--settings`)

4. **Orchestration** (`internal/core/spawn/orchestrate.go`)
   - Pass agent to `ResumeRequest` for resume operations

### Copilot Defaults

Created new defaults at `internal/install/defaults/copilot/`:
- `cortex.yaml` - Config with `agent: copilot` for all roles
- `CONFIG_DOCS.md` - Copilot-specific configuration guide
- `prompts/architect/KICKOFF.md` - Full architect workflow + MCP tool docs
- `prompts/ticket/{work,debug,research,chore}/KICKOFF.md` - Workflow guidance + MCP tools
- `prompts/ticket/{work,debug,research,chore}/APPROVE.md` - Approval prompts

Key difference: Since Copilot doesn't support `--system-prompt`, all MCP tool documentation and workflow guidance is included in KICKOFF.md files.

### Install & Upgrade

1. **Install Logic** (`internal/install/install.go`)
   - Added `setupCopilotDefaults()` function
   - Install Copilot defaults alongside claude-code during `cortex init`

2. **Defaults Upgrade** (`cmd/cortex/commands/defaults_upgrade.go`)
   - Refactored to handle both `claude-code` and `copilot` config directories
   - Loops over all configs for dry-run and apply operations

3. **MCP Tool Update** (`internal/daemon/mcp/tools_architect.go`)
   - Updated `getCortexConfigDocs()` to handle naming:
     - `claude` → `claude-code` directory
     - `copilot` → `copilot` directory (no `-code` suffix)

## Testing

- ✅ `make lint` - No issues
- ✅ `make test` - All tests pass
- ✅ `make build` - Compiles successfully

## Backward Compatibility

- Existing `claude` projects continue to work unchanged
- Default agent remains `claude` when not specified
- Copilot doesn't support lifecycle hooks via `--settings` (documented limitation)