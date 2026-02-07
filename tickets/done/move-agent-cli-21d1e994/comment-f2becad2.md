---
id: f2becad2-6229-430e-b581-483a3ec6f1da
author: claude
type: ticket_done
created: 2026-01-28T07:44:39.152494Z
---
## Summary

Moved all agent-specific CLI flags (`--permission-mode`, `--allowedTools`) from hardcoded spawn logic into configurable `agent_args` in `.cortex/cortex.yaml`. This makes Cortex agent-agnostic — swapping agents no longer requires source changes.

## Changes Made

### 1. `internal/core/spawn/launcher.go`
- Removed `PermissionMode` and `AllowedTools` fields from `LauncherParams` struct
- Removed rendering logic for these fields in `buildLauncherScript()`
- These flags now flow exclusively through the existing `AgentArgs` field (shell-quoted via `shellQuote()`)

### 2. `internal/core/spawn/spawn.go`
- Replaced the `switch req.AgentType` block that set `AllowedTools` (architect) and `PermissionMode` (ticket agent) with a simpler block that only sets Cortex-internal concerns: `ReplaceSystemPrompt` for architect, `EnvVars` for ticket agents
- Added `AgentArgs []string` field to `ResumeRequest` struct
- Updated `Resume()` to pass `req.AgentArgs` instead of hardcoded `PermissionMode: "plan"`

### 3. `internal/core/spawn/orchestrate.go`
- Added `AgentArgs: projectCfg.AgentArgs.Ticket` to the `ResumeRequest` in the resume path, ensuring resumed sessions receive the same agent args as fresh spawns

### 4. `internal/install/install.go`
- Added `agent_args` defaults to the `cortex init` config template:
  - Architect: `--allowedTools mcp__cortex__listTickets,mcp__cortex__readTicket`
  - Ticket: `--permission-mode plan`, `--allow-dangerously-skip-permissions`, `--allowedTools mcp__cortex__readTicket`

### 5. `internal/core/spawn/spawn_test.go`
- Updated 4 test functions to use `AgentArgs` instead of `PermissionMode`/`AllowedTools`
- Updated assertions to match shell-quoted output format (`'--flag' 'value'`)
- Inverted assertion in `TestSpawn_TicketAgent_Success` to verify `--permission-mode` is absent when no `AgentArgs` provided

## Key Decisions

- **`ReplaceSystemPrompt` stays in spawn code**: During merge, resolved a conflict with a new `ReplaceSystemPrompt` field added in main. This is a Cortex-internal concern (controls `--system-prompt` vs `--append-system-prompt`) and correctly stays in spawn logic, unlike agent-specific flags.
- **Switch block preserved (reduced)**: Kept a `switch req.AgentType` in spawn.go but only for Cortex-internal concerns (`ReplaceSystemPrompt`, `EnvVars`), not agent CLI flags.
- **Shell quoting**: `AgentArgs` are shell-quoted via `shellQuote()`, producing `'--flag' 'value'` format. Tests updated accordingly.

## Verification
- `make test` — All unit tests pass
- `make lint` — 0 issues
- `make build` — Compiles cleanly
- Merge conflict resolved and verified post-merge

## Files Modified
- `internal/core/spawn/launcher.go`
- `internal/core/spawn/spawn.go`
- `internal/core/spawn/orchestrate.go`
- `internal/core/spawn/spawn_test.go`
- `internal/install/install.go`