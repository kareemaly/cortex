---
id: e3498131-7a0c-4b73-81a6-0fb487c17088
author: claude
type: review_requested
created: 2026-02-10T08:22:02.508315Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/e0fb6fd6-3158-401a-aecf-6b3b0a6c9f05
        commit: efadecc
---
## Changes

Cleans up MCP tool boundaries between meta and architect agents:

### Removals
- **Meta**: Removed `listTickets`, `readTicket`, `listDocs`, `readDoc` tools — ticket/doc visibility belongs to the architect layer, not meta
- **Architect**: Removed `getCortexConfigDocs` tool, handler, and unused imports (`os`, `path/filepath`, `install`) — config management belongs to meta
- **Types**: Removed `GetCortexConfigDocsInput` and `GetCortexConfigDocsOutput` structs

### Enforcement
- **Meta `listSessions`**: Added explicit `project_path` required validation before `validateProjectPath`, matching the pattern used by `handleMetaSpawnArchitect`

### Documentation
- Updated meta `SYSTEM.md`: Removed "Cross-Project Awareness" section
- Updated architect `SYSTEM.md`: Removed `getCortexConfigDocs` from tool list and config guidance line
- Updated `CLAUDE.md`: Removed 4 ticket/doc rows from meta table, removed `getCortexConfigDocs` from architect table, updated `listSessions` description to note `project_path` is required

### Verification
- `make build` — compiles successfully
- `make test` — all tests pass
- `make lint` — 0 issues