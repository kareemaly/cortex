# MCP Integration

Wire lifecycle hooks and session spawning into MCP tools.

## Context

The MCP server has all tools implemented but some are stubbed or missing hook integration. The lifecycle, tmux, and project config packages are ready but not connected.

See `DESIGN.md` for:
- MCP tools behavior (lines 171-202)
- Hook execution on tools (lines 188-192)
- Agent spawning (lines 356-378)
- MCP config format (lines 383-396)

Existing packages to integrate:
- `internal/lifecycle` - hook execution
- `internal/tmux` - session management
- `internal/project/config` - load hook definitions
- `internal/git` - get git_base for sessions

## Requirements

Update `internal/daemon/mcp/` to:

1. **Load Project Config**
   - Use `project/config.LoadFromPath()` at server init
   - Make hook definitions available to tools

2. **Wire pickupTicket**
   - Execute `on_pickup` hooks after marking ticket in progress
   - Pass template variables (ticket_id, ticket_slug, ticket_title)
   - Return hook results in response

3. **Wire submitReport**
   - Execute `on_submit` hooks after updating report
   - Return hook results (success/failure, stdout)

4. **Wire approve**
   - Execute `on_approve` hooks with commit_message
   - Only move to done if hooks succeed
   - Return hook results

5. **Wire spawnSession**
   - Create tmux window using `internal/tmux`
   - Generate MCP config file for the session
   - Run `claude --mcp-config <path> --prompt <ticket-content>`
   - Record git_base using `internal/git`
   - Add session to ticket

## Verification

```bash
make build   # Builds successfully
make test    # Tests pass
make lint    # No lint errors

# Manual test with cortexd mcp
echo '{"method":"listTickets"}' | cortexd mcp
```

## Notes

- Hook failures should return error to agent, not crash
- Template variable errors should be reported clearly
- spawnSession creates detached tmux window (agent runs independently)
- MCP config should use CORTEX_TICKET_ID env var
