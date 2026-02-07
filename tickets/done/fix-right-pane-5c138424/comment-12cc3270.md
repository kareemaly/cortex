---
id: 12cc3270-a42c-46e0-bd96-e634988c63d6
author: claude
type: ticket_done
created: 2026-01-26T18:13:21.807123Z
---
## Summary
Fixed the right companion pane working directory in worktree mode. When spawning a ticket agent in worktree mode, both panes were using the worktree path as their working directory. The right pane runs `cortex show`, which needs the project root to locate `.cortex/` and resolve the ticket. Added a `companionWorkingDir` parameter to `SpawnAgent()` and `SpawnArchitect()` so each pane can have its own working directory.

## Key Decisions
- Added a separate `companionWorkingDir` parameter rather than hardcoding the project path, keeping the API flexible for future use cases.
- In `Resume()`, both working dirs use `req.ProjectPath` since resume already operates from the project root.
- In `spawnInTmux()`, the agent pane uses `workingDir` (worktree path in worktree mode) while the companion pane uses `req.ProjectPath` (project root).

## Files Modified
- `internal/core/spawn/spawn.go` — Updated `TmuxManagerInterface` with new parameter; updated `spawnInTmux()` and `Resume()` call sites
- `internal/tmux/command.go` — Updated `SpawnAgent()` and `SpawnArchitect()` to accept and use `companionWorkingDir` for `SplitWindowHorizontal`
- `internal/core/spawn/spawn_test.go` — Updated mock signatures
- `internal/tmux/integration_test.go` — Updated integration test call sites

## Verification
- `make build` — compiles cleanly
- `make test` — all unit tests pass
- `make lint` — 0 issues