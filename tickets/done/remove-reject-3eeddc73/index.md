---
id: 3eeddc73-f07e-439b-a92b-1528a5696909
title: Remove reject feature and all related code
type: work
created: 2026-02-05T06:32:17.085292Z
updated: 2026-02-05T06:40:02.922941Z
---
## Objective

Remove the reject workflow entirely from Cortex. This simplifies the approval flow to just approve/conclude.

## Scope

Remove all reject-related code:

### Prompts
- Delete `prompts/ticket/work/REJECT.md` from embedded defaults
- Delete from `~/.cortex/defaults/claude-code/` if installed
- Remove `StageReject` constant from `internal/prompt/`

### API
- Remove reject endpoint from `internal/daemon/api/sessions.go`
- Remove route registration in `internal/daemon/api/server.go`
- Update SDK client in `internal/cli/sdk/client.go`

### MCP Tools
- Check if any MCP tools reference reject functionality

### TUI
- Remove any reject buttons/actions from ticket detail view
- Update any reject-related keybindings

### Tests
- Update or remove tests that cover reject functionality

### Documentation
- Update CLAUDE.md if reject is mentioned
- Update any comments referencing reject workflow

## Acceptance Criteria

- No references to "reject" remain in codebase (except git history)
- Build passes: `make build`
- Lint passes: `make lint`
- Tests pass: `make test`
- Approval flow still works (approve → conclude)