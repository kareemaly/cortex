---
id: ff1a46e0-13fe-4bba-a663-3c652646239c
title: Implement Copilot CLI as an agent type
type: work
created: 2026-02-05T10:04:18.081099Z
updated: 2026-02-05T10:23:00.876686Z
---
## Summary

Add GitHub Copilot CLI as a supported agent type alongside Claude, enabling multi-model flexibility.

## Context

Research ticket `83c6289c` confirmed high feasibility. The standalone `copilot` CLI has compatible MCP support and similar architecture to Claude Code.

## Requirements

### 1. Add Copilot agent type
- Add `AgentCopilot AgentType = "copilot"` in `internal/project/config/config.go`
- Update `Validate()` to accept `copilot`

### 2. Launcher script for Copilot
- Add `buildCopilotLauncherScript()` in `internal/core/spawn/launcher.go`
- Flag mapping:
  - `--additional-mcp-config` (MCP config path)
  - `--yolo` (required for automation)
  - `--no-custom-instructions` (ignore AGENTS.md files)
  - `--resume <id>` (session resume)

### 3. Skip SYSTEM.md for Copilot agents
- Copilot doesn't support `--system-prompt` CLI flag
- Modify `internal/core/spawn/spawn.go` to skip SYSTEM.md loading when agent=copilot
- All workflow guidance goes in KICKOFF.md instead

### 4. Create Copilot defaults
- New embedded folder: `internal/install/defaults/copilot/`
- Structure:
  ```
  copilot/
  ├── cortex.yaml           # agent: copilot, args: [--yolo]
  ├── CONFIG_DOCS.md
  └── prompts/
      ├── architect/
      │   └── KICKOFF.md    # Full workflow + MCP tool docs
      └── ticket/
          ├── work/KICKOFF.md
          ├── debug/KICKOFF.md
          ├── research/KICKOFF.md
          └── chore/KICKOFF.md
  ```
- KICKOFF.md must include MCP tool inventory and workflow guidance (since no SYSTEM.md)

### 5. Update defaults upgrade
- Modify `internal/install/` to install copilot defaults alongside claude-code

## Implementation Notes

- Project config uses `extend: ~/.cortex/defaults/copilot` to select Copilot agent
- MCP config format is compatible (both use `mcpServers` JSON structure)
- Copilot supports multiple models via `--model` flag (claude-sonnet-4.5, gpt-5.2, etc.)

## Out of Scope
- Model selection UI (users configure via `args` in cortex.yaml)
- Hybrid agent support (one agent type per project for now)