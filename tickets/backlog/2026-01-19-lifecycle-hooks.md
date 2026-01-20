# Lifecycle Hooks

Implement the lifecycle hook execution framework.

## Context

Hooks are shell commands that run at key points in the ticket lifecycle. They're defined in `.cortex/cortex.yaml` and executed by MCP tools.

See `DESIGN.md` for:
- Lifecycle hooks config (lines 228-260)
- Hook behavior (lines 255-259)
- Template variables (lines 263-269)
- Hook response format (lines 206-226)

## Requirements

Create `internal/lifecycle/` package that:

1. **Hook Types**
   - `on_pickup` - runs when agent starts work
   - `on_submit` - runs when agent submits report
   - `on_approve` - runs when ticket is approved

2. **Execution**
   - Run shell commands sequentially
   - Capture stdout for each command
   - Stop on first non-zero exit code
   - Return structured result (success, hook outputs, exit codes)

3. **Template Variables**
   - `{{ticket_id}}` - ticket UUID
   - `{{ticket_slug}}` - slugified title
   - `{{ticket_title}}` - full title
   - `{{commit_message}}` - only for on_approve

4. **Interface**
   - Accept hook definitions and template values
   - Return results per DESIGN.md hook response format
   - No config loading - that's the project-config ticket's job

## Verification

```bash
make build   # Builds successfully
make test    # Tests pass
make lint    # No lint errors
```

## Notes

- This package executes hooks, doesn't load config
- MCP integration is a separate ticket
- Commands run in project directory context
- Handle missing commands gracefully
