---
id: 621e10a1-c9e5-481f-a8d2-9eba771b0a6b
author: claude
type: done
created: 2026-02-05T10:23:00.874777Z
---
## Summary

Implemented GitHub Copilot CLI as a supported agent type alongside Claude, enabling multi-model flexibility in Cortex.

## Changes Made

### Core Implementation
- Added `AgentCopilot` constant to `internal/project/config/config.go` with updated validation
- Made launcher agent-aware in `internal/core/spawn/launcher.go` with Copilot-specific command builder using `--yolo`, `--no-custom-instructions`, and `--additional-mcp-config` flags
- Updated spawn flow in `internal/core/spawn/spawn.go` to skip SYSTEM.md and settings for Copilot agents
- Updated orchestration in `internal/core/spawn/orchestrate.go` to pass agent to ResumeRequest

### New Files Created
- `internal/install/defaults/copilot/` directory with:
  - `cortex.yaml` - Config with `agent: copilot` for all roles
  - `CONFIG_DOCS.md` - Copilot-specific configuration guide
  - `prompts/architect/KICKOFF.md` - Full architect workflow + MCP tool docs
  - `prompts/ticket/{work,debug,research,chore}/KICKOFF.md` and `APPROVE.md`

### Install & Upgrade
- Added `setupCopilotDefaults()` to `internal/install/install.go`
- Updated `cmd/cortex/commands/defaults_upgrade.go` to handle both `claude-code` and `copilot` configs
- Updated `getCortexConfigDocs` MCP tool for correct directory naming

### Documentation
- Updated `CLAUDE.md` to list available agent types and add agent defaults path

## Commits
1. `327a7c9` - feat: add GitHub Copilot CLI as agent type
2. `44fa9cc` - docs: document copilot agent type in CLAUDE.md

## Testing
- All tests pass (`make test`)
- Build succeeds (`make build`)
- Lint passes (`make lint`)
- Pre-push hooks pass