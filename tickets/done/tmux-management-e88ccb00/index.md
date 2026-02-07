---
id: e88ccb00-5102-4ecf-b784-94629fca5733
title: Tmux Management
type: ""
created: 2026-01-20T13:06:54Z
updated: 2026-01-20T13:06:54Z
---
Implement tmux session and window management for agent sessions.

## Context

Each project has a tmux session. Each ticket gets a window within that session. The architect gets window 0.

See `DESIGN.md` for:
- Tmux naming convention (lines 154-159)
- Session types (lines 57-78)
- Agent spawning examples (lines 356-378)

## Requirements

Create `internal/tmux/` package that:

1. **Session Management**
   - Create session if not exists (named after project)
   - Check if session exists
   - Kill session

2. **Window Management**
   - Create window with name (slugified ticket title, max 20 chars)
   - Window 0 reserved for architect
   - Focus/attach to window
   - Kill window
   - Check if window exists

3. **Command Execution**
   - Run command in window (for spawning agents)
   - Send keys to window

4. **Utilities**
   - List windows in session
   - Get active window

## Verification

```bash
make build   # Builds successfully
make test    # Tests pass (may need integration tests with tmux)
make lint    # No lint errors
```

## Notes

- Use `os/exec` to shell out to tmux commands
- Handle cases where tmux isn't installed gracefully
- Window names should match ticket slugs for easy identification
- Tests may need to be skipped in CI if tmux unavailable

## Implementation

### Commits Pushed
- `cb659e5` feat: add tmux session and window management package
- `e7aa603` Merge branch 'ticket/2026-01-19-tmux-management'

### Key Files Changed
- `internal/tmux/errors.go` - Custom error types (NotInstalledError, SessionNotFoundError, WindowNotFoundError, CommandError)
- `internal/tmux/tmux.go` - Manager struct with NewManager() and Available()
- `internal/tmux/session.go` - Session management (SessionExists, CreateSession, KillSession, AttachSession)
- `internal/tmux/window.go` - Window management (CreateWindow, CreateArchitectWindow, KillWindow, FocusWindow, ListWindows, GetWindowByName)
- `internal/tmux/command.go` - Command execution (RunCommand, SendKeys, SpawnAgent, SpawnArchitect)
- `internal/tmux/tmux_test.go` - Unit tests for error types and constants
- `internal/tmux/integration_test.go` - Integration tests with `//go:build integration` tag

### Important Decisions
- Manager struct holds tmux binary path, returned by NewManager() which fails with NotInstalledError if tmux unavailable
- CreateSession is idempotent (no-op if session exists)
- Window operations use `session:window` target format
- Integration tests use build tag and skip in CI via environment variable check
- SpawnAgent/SpawnArchitect are high-level helpers that create session if needed, create window, and run command

### Scope Changes
None - implementation matches original ticket requirements