# Split Pane Layout on Spawn

## Problem

When spawning architect or ticket session, only one pane is created for the agent.

## Expected Behavior

When spawning, create a split layout:
- 50% left pane: agent (claude)
- 50% right pane:
  - For architect: `cortex kanban`
  - For ticket: `cortex show` (will use CORTEX_TICKET_ID env var)

## Verification

```bash
make lint
make test
make build
make test-integration
```

## Implementation

### Commits Pushed

- `43b55c8` feat: add split pane layout when spawning agent sessions

### Key Files Changed

| File | Change |
|------|--------|
| `internal/tmux/pane.go` | **NEW** - Added `SplitWindowHorizontal`, `RunCommandInPane`, `SelectPane` methods |
| `internal/tmux/command.go` | Updated `SpawnAgent` and `SpawnArchitect` to accept `companionCommand` parameter |
| `internal/core/spawn/spawn.go` | Updated `TmuxManagerInterface` and callers to pass companion commands |
| `internal/core/spawn/spawn_test.go` | Updated mock with new interface signatures |
| `internal/tmux/integration_test.go` | Added tests for pane methods and split pane spawning |

### Important Decisions

- Companion command for ticket sessions uses `CORTEX_TICKET_ID` env var prefix so `cortex show` can auto-detect the ticket
- Focus returns to left pane (agent) after creating split layout
- Split uses horizontal layout (`-h` flag) for side-by-side panes

### Scope Changes

None - implemented as planned.
