---
id: a7ff0b61-0198-4825-be66-0e73d1a2313e
title: MCP Integration
type: ""
created: 2026-01-20T14:16:36Z
updated: 2026-01-20T14:16:36Z
---
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

## Implementation

### Commits Pushed
- `3a17039` feat: integrate lifecycle hooks and implement spawnSession in MCP tools

### Key Files Changed
- `internal/daemon/mcp/server.go` - Extended Config with ProjectPath/TmuxSession, added projectConfig and lifecycle executor
- `internal/daemon/mcp/types.go` - Added HookResultOutput, HooksExecutionOutput, updated output types with Hooks field
- `internal/daemon/mcp/helpers.go` - New file with hook conversion utilities (convertHookConfigs, buildTemplateVars, getHooksForType, convertExecutionResult)
- `internal/daemon/mcp/tools_ticket.go` - Wired on_pickup, on_submit, on_approve hooks into handlers
- `internal/daemon/mcp/tools_architect.go` - Full spawnSession implementation with tmux/MCP config generation
- `internal/daemon/mcp/tools_test.go` - Updated and added tests for spawnSession

### Important Decisions
1. **Hook failure behavior**: on_pickup and on_submit hooks log failures but don't fail the operation; on_approve hooks must succeed before moving ticket to done
2. **Permission mode**: Uses `--permission-mode plan` for ticket sessions (following cortex0 pattern for controlled execution)
3. **MCP config location**: Generated at `/tmp/cortex-mcp-{ticket_id}.json` with cortexd command and environment variables
4. **Prompt format**: Passes ticket title and body as positional argument to claude, instructs agent to use cortex MCP tools

### Scope Changes
- Original ticket mentioned `--prompt` flag for claude, but claude CLI uses positional argument for prompts (fixed during implementation)
- Added `--permission-mode plan` based on comparison with cortex0 approach (not in original requirements)