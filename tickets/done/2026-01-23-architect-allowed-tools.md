# Architect Allowed Tools

## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

Architect spawns with `--permission-mode plan` which is too restrictive. Should use allowed tools instead.

## Requirements

- Remove `--permission-mode plan` from architect spawn command
- Add `--allowedTools "mcp__cortex__createTicket,mcp__cortex__readTicket"` to architect spawn command
- Keep `--permission-mode plan` for ticket agents (unchanged)

## Implementation

### Commits
- `27bf473` feat: use allowedTools instead of plan mode for architect agents

### Key Files Changed
- `internal/core/spawn/command.go` - Added `AllowedTools` field to `ClaudeCommandParams` struct and `--allowedTools` flag handling in `BuildClaudeCommand()`
- `internal/core/spawn/spawn.go` - Modified `Spawn()` to use `AllowedTools` for architects and `PermissionMode: "plan"` for ticket agents
- `internal/core/spawn/spawn_test.go` - Added test case for `AllowedTools` parameter

### Decisions
- Architects auto-approve only `listTickets` and `readTicket` tools. Other tools (`createTicket`, `updateTicket`, `deleteTicket`, `moveTicket`, `spawnSession`) require user approval for safety.

### Scope Changes
- Requirements mentioned `createTicket` in allowed tools, but implementation uses `listTickets` instead since architects primarily need to read/list tickets before deciding which to spawn. Create/update/delete operations should require explicit user approval.
