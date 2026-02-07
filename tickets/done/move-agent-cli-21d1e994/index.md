---
id: 21d1e994-4c6a-497e-a552-9cde44648e44
title: Move Agent CLI Flags to Config agent_args with Sensible Defaults
type: ""
created: 2026-01-28T06:52:37.434604Z
updated: 2026-01-28T07:44:39.153997Z
---
## Summary

Make Cortex agent-agnostic by moving all agent-specific CLI flags (permission mode, allowed tools, etc.) from hardcoded spawn logic into configurable `agent_args` in `.cortex/cortex.yaml`. Update `cortex init` to generate sensible defaults per agent type.

## Motivation

Preparing for multi-agent support. The spawn code currently hardcodes Claude-specific flags (`--permission-mode plan`, `--allowedTools`). These should live in user-editable config so swapping agents doesn't require Cortex source changes.

## Changes

### 1. Update `cortex init` defaults

Generate `agent_args` in `.cortex/cortex.yaml` with these defaults:

```yaml
agent: claude
agent_args:
  architect:
    - "--allowedTools"
    - "mcp__cortex__listTickets,mcp__cortex__readTicket"
  ticket:
    - "--permission-mode"
    - "plan"
    - "--allow-dangerously-skip-permissions"
    - "--allowedTools"
    - "mcp__cortex__readTicket"
```

### 2. Remove hardcoded flags from spawn code

In `internal/core/spawn/spawn.go`:
- Remove `LauncherParams.PermissionMode` field and its handling
- Remove `LauncherParams.AllowedTools` field and its handling
- Remove the `switch AgentType` block that sets these per agent type
- Launcher builder just appends `AgentArgs` from config to the command

### 3. Update launcher script generation

In `internal/core/spawn/launcher.go`:
- Remove `PermissionMode` and `AllowedTools` from `LauncherParams` struct
- Remove template logic that renders these fields
- `AgentArgs` already gets appended — that's the only mechanism now

### What stays in spawn code (Cortex-level concerns)

- Env vars (`CORTEX_TICKET_ID`, `CORTEX_PROJECT`)
- MCP config (`--mcp-config`)
- Settings/hooks (`--settings`)
- System prompt (`--append-system-prompt`)
- Session ID (`--session-id`)
- Companion pane commands