---
id: a5bce206-473e-46ec-a744-b902fe86ea66
title: Restructure Prompts, Config Schema, and Ticket Model for Type-Based Architecture
type: ""
created: 2026-01-28T09:04:41.243507Z
updated: 2026-01-28T09:37:14.837024Z
---
## Summary

Restructure the prompt system, project config, and ticket model to support type-based ticket workflows. This replaces the flat prompt layout and config schema with a hierarchical structure organized by role (architect/ticket) and ticket type (work, and future types like investigation/debugging).

## Current State

**Prompts** (`.cortex/prompts/`): 6 flat files — `architect.md`, `ticket-system.md`, `ticket.md`, `ticket-worktree.md`, `approve.md`, `approve-worktree.md`

**Config** (`cortex.yaml`):
```yaml
agent: claude
agent_args:
  architect: [...]
  ticket: [...]
```

**Ticket model**: No `type` field.

**Prompt loading** (`internal/prompt/`): Hardcoded path functions per file. Worktree handled via separate template files.

## Target State

### 1. Prompt folder structure

```
.cortex/prompts/
  architect/
    SYSTEM.md          # was architect.md
    KICKOFF.md         # dynamic project/ticket list prompt
  ticket/
    work/
      SYSTEM.md        # was ticket-system.md
      KICKOFF.md       # merged ticket.md + ticket-worktree.md (accepts worktree boolean)
      APPROVE.md       # merged approve.md + approve-worktree.md (accepts worktree boolean)
```

- SYSTEM = identity, rules, capabilities (the system prompt)
- KICKOFF = the assignment/task prompt sent at session start
- APPROVE = approval workflow prompt sent when work is accepted
- Worktree-specific content is merged into KICKOFF.md and APPROVE.md via a boolean template variable (e.g., `{{.IsWorktree}}`) instead of separate files
- Delete old flat files: `ticket.md`, `ticket-worktree.md`, `approve.md`, `approve-worktree.md`, `architect.md`, `ticket-system.md`

### 2. Config schema (`cortex.yaml`)

```yaml
architect:
  agent: claude
  args: [--flags, --here]

ticket:
  work:
    agent: claude
    args: [--flags, --here]
  # future types: investigation, debugging, etc.
```

- Replace flat `agent` and `agent_args` fields with nested `architect:` and `ticket.<type>:` sections
- Each section has `agent` (AgentType) and `args` ([]string)
- No backward compatibility needed — break clean

### 3. Ticket model

- Add `Type` string field to the ticket struct (default: `work`)
- Expose in create/read APIs and MCP `createTicket` tool (optional parameter, defaults to `work`)
- The type determines which prompt folder under `ticket/` is used

### 4. Prompt loading (`internal/prompt/`)

- Replace hardcoded per-file path functions with type-based resolution:
  - `ArchitectPromptPath(stage string)` → `.cortex/prompts/architect/{stage}.md`
  - `TicketPromptPath(ticketType string, stage string)` → `.cortex/prompts/ticket/{type}/{stage}.md`
- Update `TicketVars` to include `IsWorktree bool` field
- Remove old path functions: `TicketPath()`, `TicketWorktreePath()`, `ApprovePath()`, `ApproveWorktreePath()`, etc.

### 5. Spawn logic (`internal/core/spawn/`)

- Update architect spawn to read config from `architect.agent` / `architect.args`
- Update ticket spawn to read config from `ticket.<type>.agent` / `ticket.<type>.args` using the ticket's type
- Update prompt assembly to use new path resolution and pass `IsWorktree` as template variable
- Update approval flow to use `APPROVE.md` with `IsWorktree` boolean

### 6. Init/install

- Update `cortex init` to generate the new folder structure and default prompt files
- Update default config template to use the new schema shape

## Acceptance Criteria

- [ ] Ticket struct has `Type` field, defaults to `work`, persisted in JSON
- [ ] `createTicket` MCP tool and API accept optional `type` parameter
- [ ] `cortex.yaml` uses new nested schema (`architect:` and `ticket.<type>:`)
- [ ] Config parsing reads agent/args from the new structure
- [ ] Prompts live in `prompts/architect/` and `prompts/ticket/work/` with SYSTEM/KICKOFF/APPROVE naming
- [ ] Worktree-specific content merged into templates via `{{.IsWorktree}}` conditional blocks
- [ ] Old flat prompt files are removed
- [ ] Spawn logic resolves agent config and prompts by ticket type
- [ ] `cortex init` generates the new folder structure
- [ ] Existing tests updated, all pass (`make test`)
- [ ] Architect and ticket agent sessions work end-to-end with the new structure