# Tmux Management

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
