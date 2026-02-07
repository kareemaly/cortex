---
id: ae7d5caa-a03d-4d05-8a65-c52c10214488
title: Implement multiple ticket types (debug, research, chore)
type: work
created: 2026-02-05T06:47:04.272702Z
updated: 2026-02-05T07:08:06.241694Z
---
## Objective

Add support for multiple ticket types beyond `work`. Each type has its own prompts and agent configuration, enabling specialized workflows.

## New Types

| Type | Purpose | Key Behavior |
|------|---------|--------------|
| `debug` | Root cause analysis | Systematic investigation, document findings, then fix |
| `research` | Exploration | Read-only, brainstorm with user, no file changes, document in comments |
| `chore` | Maintenance | Minimal ceremony, direct execution |

## Implementation

### 1. Prompts — Create directories in `internal/install/defaults/claude-code/prompts/ticket/`

**debug/**
- `SYSTEM.md`: MCP tools. Systematic debugging workflow: reproduce → isolate → hypothesize → verify → fix. Add comments documenting findings.
- `KICKOFF.md`: Ticket details. "Investigate this issue. Document reproduction steps. Find root cause before fixing. Add ticket comments with observations."
- `APPROVE.md`: "Fix approved. Commit with root cause explanation. Conclude with summary of cause and resolution."

**research/**
- `SYSTEM.md`: MCP tools. **Read-only exploration mode. Do NOT modify files.** Brainstorm with user. Add ticket comments with findings.
- `KICKOFF.md`: Ticket details. "Research this topic. Explore codebase, ask questions, brainstorm. Document findings as ticket comments. No file changes."
- `APPROVE.md`: "Research complete. Conclude with summary of findings and recommendations."

**chore/**
- `SYSTEM.md`: MCP tools. Maintenance mode. Direct execution, minimal process.
- `KICKOFF.md`: Ticket details. "Complete this maintenance task. Keep changes focused. Request review when done."
- `APPROVE.md`: "Approved. Commit and conclude."

### 2. Config — Update `internal/install/defaults/claude-code/cortex.yaml`

```yaml
ticket:
  work:
    agent: claude
  debug:
    agent: claude
    args: ["--dangerously-skip-permissions"]
    allowedTools: ["mcp__cortex__*"]
  research:
    agent: claude
    args: ["--dangerously-skip-permissions"]
    allowedTools: ["mcp__cortex__*"]
  chore:
    agent: claude
    args: ["--dangerously-skip-permissions"]
    allowedTools: ["mcp__cortex__*"]
```

### 3. Config Schema — Update `internal/project/config/config.go`

- Add `AllowedTools []string` field to `RoleConfig`
- Ensure spawn logic passes allowed tools to agent

### 4. Spawn Logic — Update `internal/core/spawn/spawn.go`

- Pass `--dangerously-skip-permissions` arg when configured
- Pass `--allowedTools` when configured (check claude CLI flag name)

### 5. TUI — Update kanban card rendering

- Add type badge to ticket cards: `[debug] Fix null pointer...`
- Location: `internal/cli/tui/kanban/column.go` or card rendering

### 6. Validation

- Validate ticket type exists in config at creation time
- Return clear error for unknown types

## Acceptance Criteria

- [ ] All four types have prompt directories with SYSTEM, KICKOFF, APPROVE
- [ ] Config schema supports `args` and `allowedTools` per type
- [ ] Spawn passes configured args to agent
- [ ] Kanban shows type badges
- [ ] `make build && make lint && make test` pass
- [ ] Manual test: create and spawn each ticket type