# Architect Should Open in Project Directory

## Problem

When spawning architect via `cortex architect`, the tmux window opens in the wrong directory. It opens in `~/projects/cortex1` instead of the current project directory.

## Expected Behavior

Architect session should start in the project directory that was current when `cortex architect` was invoked.

## Verification

```bash
make lint
make test
make build
make test-integration
```

Then test:
1. `cd ~/projects/some-other-project`
2. `cortex architect`
3. Verify tmux window opens in `~/projects/some-other-project`

## Implementation

### Commits Pushed
- `77093b5` fix: use project working directory when spawning tmux sessions
- `8f039c5` Merge branch 'ticket/2026-01-23-architect-working-directory'

### Key Files Changed
- `internal/tmux/session.go` - Added `workingDir` param to `CreateSession`, uses `-c` flag
- `internal/tmux/window.go` - Added `workingDir` param to `CreateWindow`, uses `-c` flag
- `internal/tmux/command.go` - Updated `SpawnAgent` and `SpawnArchitect` to pass working directory; for existing sessions, prepends `cd` to command
- `internal/core/spawn/spawn.go` - Updated `TmuxManagerInterface` and `spawnInTmux` to pass `ProjectPath`
- `internal/core/spawn/spawn_test.go` - Updated mock with new signatures and `lastWorkingDir` field
- `internal/tmux/integration_test.go` - Updated all calls with empty string for backwards compatibility

### Important Decisions
- For new sessions: Use tmux's `-c` flag to set working directory when creating sessions/windows
- For existing sessions: Prepend `cd <workingDir> &&` to the agent command since tmux `-c` flag only works at creation time
- Pass `ProjectPath` from `SpawnRequest` through the entire chain to tmux manager

### Scope Changes
None - implemented exactly as planned
