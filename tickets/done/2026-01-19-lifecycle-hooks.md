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

## Implementation

### Commits
- `e7ec3a9` feat: add lifecycle hooks package for shell command execution at ticket lifecycle points

### Key Files Changed
- `internal/lifecycle/errors.go` - Error types (ExecutionError, TemplateError, InvalidVariableError) with Is* helpers
- `internal/lifecycle/hooks.go` - Core types (Executor, HookDefinition, ExecutionResult, TemplateVars) and CommandRunner interface
- `internal/lifecycle/template.go` - Template variable expansion using regex, validation for hook-specific variables
- `internal/lifecycle/hooks_test.go` - 20 test functions with mock runner for comprehensive coverage

### Key Decisions
- Used `sh -c` for command execution to support shell features (pipes, redirects)
- CommandRunner interface allows mock injection for testing without actual shell execution
- Unknown template variables are left unchanged (graceful degradation) rather than erroring
- Follows error pattern from `internal/git/errors.go` for consistency
- Template validation ensures `{{commit_message}}` only used in `on_approve` hooks

### Scope
No changes from original ticket scope
