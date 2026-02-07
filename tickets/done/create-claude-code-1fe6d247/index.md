---
id: 1fe6d247-a2bf-43b7-a74f-9a73320fa377
title: Create claude-code config docs
type: work
created: 2026-02-03T08:28:06.918139Z
updated: 2026-02-03T08:40:44.310522Z
---
# Overview

Create an embedded markdown document containing all configuration guidance for claude-code agent type. This will be returned by the `getCortexConfigDocs` MCP tool (future ticket).

## Location

```
internal/install/defaults/claude-code/CONFIG_DOCS.md
```

## Content

The document should cover:

### 1. Project Config Schema
- `.cortex/cortex.yaml` fields
- `name`, `extend`, `architect`, `ticket`, `git` sections
- Example YAML with comments

### 2. Prompt Structure
- Table of all prompts and their purposes
- `architect/SYSTEM.md`, `architect/KICKOFF.md`
- `ticket/work/SYSTEM.md`, `KICKOFF.md`, `APPROVE.md`, `REJECT.md`

### 3. Customizing Prompts
- `cortex eject` usage and examples
- Auto-discovery explanation (`.cortex/prompts/` overrides defaults)

### 4. Template Variables
- Table of available variables per context
- `{{.ProjectName}}`, `{{.TicketList}}`, `{{.TicketTitle}}`, `{{.TicketBody}}`
- Worktree variables: `{{.IsWorktree}}`, `{{.WorktreePath}}`, `{{.WorktreeBranch}}`, `{{.ProjectPath}}`

### 5. Common Customizations
- Enable git worktrees
- Restrict agent permissions
- Add test requirements to workflow
- Custom approval flow
- Project-specific context in KICKOFF

### 6. Commands Reference
- `cortex init`, `cortex eject`, `cortex architect`, `cortex kanban`

## Constraints

- Keep under 150 lines
- No upgrade/migration info (separate doc)
- No daemon/global settings
- No internal architecture details