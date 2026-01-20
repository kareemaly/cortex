# Cortex Architect Command

Implement the `cortex architect` command to start or attach to an architect session.

## Context

The architect is a singleton session per project that lives in tmux window 0. It has full MCP tools for ticket management and spawning ticket sessions.

See `DESIGN.md` for:
- Architect session description (lines 57-62)
- CLI command (line 47)
- Tmux window 0 reserved for architect (line 159)
- Agent spawning with MCP config (lines 356-378)

Existing packages to use:
- `internal/tmux` - SpawnArchitect, AttachSession, SessionExists
- `internal/project/config` - load agent type, project name
- `internal/git` - capture git_base

## Requirements

Implement `cortex architect` command that:

1. **Find or Create Architect Session**
   - Check if tmux session exists for project
   - Check if window 0 has running architect
   - If running: attach to it
   - If not: spawn new architect

2. **Spawn Architect**
   - Use tmux window 0 (reserved for architect)
   - Generate MCP config file (no --ticket-id = architect mode)
   - Run agent command with project context prompt
   - Capture git_base for session tracking

3. **Attach to Existing**
   - If architect already running, just attach to tmux window
   - Don't spawn duplicate architects

4. **Project Detection**
   - Find project root from current directory
   - Load project config for agent type and name
   - Use project name as tmux session name

## Verification

```bash
make build
make lint
make test

# Manual test
cd /some/project/with/.cortex
cortex architect  # Should spawn or attach
```

## Notes

- Architect is singleton per project (only one at a time)
- Window 0 is reserved - don't use regular CreateWindow
- Should work from any subdirectory of project
- Consider adding --detach flag to spawn without attaching
