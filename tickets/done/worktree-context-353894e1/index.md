---
id: 353894e1-2956-4430-8a2d-49ff9083d67d
title: Worktree Context Support
type: ""
created: 2026-01-24T15:58:12Z
updated: 2026-01-24T15:58:12Z
---
## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

Worktree operations in `internal/worktree/worktree.go` execute Git commands but don't support context cancellation. If an operation takes a long time, the caller cannot cancel it.

Affected methods:
- `Manager.Create()`
- `Manager.Remove()`

## Requirements

- Add `ctx context.Context` parameter to `Create()` and `Remove()` methods
- Pass context to underlying Git command execution
- Update all callers to pass context

## Implementation

### Commits

- `bd90287` feat: add context support to worktree Create and Remove methods
- `0bdd10d` Merge branch 'ticket/2026-01-24-worktree-context-support'

### Key Files Changed

| File | Changes |
|------|---------|
| `internal/worktree/worktree.go` | Added `ctx` to `Create()` and `Remove()`, use `exec.CommandContext()` |
| `internal/core/spawn/spawn.go` | Added `ctx` to `Spawn()`, `Resume()`, `Fresh()`, `cleanupOnFailure()` |
| `internal/core/spawn/spawn_test.go` | Pass `context.Background()` to spawner methods |
| `internal/daemon/api/tickets.go` | Pass `r.Context()` to spawner methods |
| `internal/daemon/api/architect.go` | Pass `r.Context()` to `Spawn()` |
| `internal/daemon/mcp/tools_architect.go` | Pass `ctx` to spawner methods |
| `internal/daemon/mcp/tools_ticket.go` | Pass `ctx` to `wm.Remove()` |

### Important Decisions

- Propagate context through the entire spawner call chain to enable cancellation at all levels
- HTTP handlers use `r.Context()` which is cancelled when the client disconnects
- MCP handlers already receive ctx from the MCP framework

### Scope Changes

None - implemented as specified in the original ticket.