---
id: bc66ecd8-de51-46c9-a2db-cd9c89ab0ed2
title: Align HTTP API Spawn with MCP Spawn
type: ""
created: 2026-01-26T16:04:42.218578Z
updated: 2026-01-26T16:29:22.58172Z
---
## Context

Project is pre-release with no users. Breaking changes are fine. No tech debt tolerance — do it right.

## Problem

The HTTP API spawn handler (`internal/daemon/api/tickets.go:293-477`) and the MCP spawn tool (`internal/daemon/mcp/tools_architect.go:215-386`) duplicate spawn orchestration logic and have diverged:

| Aspect | HTTP API (TUI) | MCP |
|--------|----------------|-----|
| **UseWorktree** | Never set (always `false`) | Set from `projectConfig.Git.Worktrees` |
| **CortexdPath** | Not passed to Spawner Dependencies | Passed from `s.config.CortexdPath` |
| **Fresh mode** | Manual `EndSession()` then `Spawn()` | Uses `spawner.Fresh()` method |
| **State matrix** | Partial validation (lines 357-404) | Complete validation (lines 285-365) |

## Solution

Extract a single spawn orchestration function that both the HTTP API and MCP call into. No duplication — one function handles:

1. State detection (existing session? ended? active? orphaned?)
2. Mode validation (normal/resume/fresh against current state)
3. Spawner construction with full dependencies (Store, TmuxManager, CortexdPath, Logger)
4. SpawnRequest building (including UseWorktree from project config)
5. Post-spawn actions (move to progress if backlog)

Both the HTTP handler and MCP tool become thin wrappers that parse their inputs and call this shared function.

## Acceptance Criteria

- [ ] Single spawn orchestration function in `internal/core/spawn/` (or similar)
- [ ] HTTP API handler is a thin wrapper calling shared function
- [ ] MCP tool is a thin wrapper calling shared function
- [ ] All spawn parameters (UseWorktree, CortexdPath, etc.) handled uniformly
- [ ] State/mode validation logic exists in exactly one place
- [ ] Post-spawn ticket move logic exists in exactly one place