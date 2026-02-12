---
id: dcef6a03-da8f-4716-83bd-91c9d99f5507
title: Clean up MCP tool boundaries between meta and architect
type: work
tags:
    - mcp
    - meta-agent
    - cleanup
created: 2026-02-10T08:14:15.642653Z
updated: 2026-02-10T08:23:16.832835Z
---
## Problem

MCP tool boundaries between meta and architect agents need tightening:

1. **`getCortexConfigDocs` on architect**: Now that the meta agent exists for config management, this tool doesn't belong on the architect. Architects don't manage Cortex configuration.

2. **Meta has ticket/doc tools it shouldn't**: Meta currently exposes `listTickets`, `readTicket`, `listDocs`, and `readDoc`. Meta should stay strictly at the project management layer — it manages projects, configs, prompts, sessions, and daemon operations. Ticket and doc visibility belongs to the architect.

3. **`project_path` optional on meta tools**: Since meta has no implicit project context, all project-scoped tools must require `project_path` explicitly. Currently `listSessions` has it as optional, which is misleading.

## Changes Required

### Architect (`internal/daemon/mcp/tools_architect.go`)
- Remove `getCortexConfigDocs` tool

### Meta (`internal/daemon/mcp/tools_meta.go`)
- Remove `listTickets`
- Remove `readTicket`
- Remove `listDocs`
- Remove `readDoc`
- Audit remaining tools and ensure `project_path` is **required** (not optional) on all project-scoped operations (e.g., `listSessions`, `spawnArchitect`)

## Acceptance criteria

- `getCortexConfigDocs` is no longer available to architect agents
- Meta agent has no ticket or doc tools
- All project-scoped meta tools require `project_path`
- Existing tests pass; update any affected test cases
- CLAUDE.md MCP tool tables updated to reflect changes