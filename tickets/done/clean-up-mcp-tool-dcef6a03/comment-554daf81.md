---
id: 554daf81-b49d-430c-b90c-d441d1e90abb
author: claude
type: done
created: 2026-02-10T08:23:16.83157Z
---
## Summary

Cleaned up MCP tool boundaries between meta and architect agents.

### Changes (6 files, +6/-99 lines)

**Removed from meta agent (`tools_meta.go`):**
- `listTickets`, `readTicket`, `listDocs`, `readDoc` — ticket/doc visibility belongs to the architect layer

**Removed from architect agent (`tools_architect.go`):**
- `getCortexConfigDocs` tool registration, handler, and 3 unused imports (`os`, `path/filepath`, `install`) — config management belongs to meta

**Removed from types (`types.go`):**
- `GetCortexConfigDocsInput` and `GetCortexConfigDocsOutput` structs

**Enforced (`tools_meta.go`):**
- Added explicit `project_path` required validation to `handleMetaListSessions`, matching the pattern used by `handleMetaSpawnArchitect`

**Updated documentation:**
- `meta/SYSTEM.md` — removed "Cross-Project Awareness" section
- `architect/SYSTEM.md` — removed `getCortexConfigDocs` from tool list and config guidance line
- `CLAUDE.md` — updated both MCP tool tables

### Verification
- `make build` ✓
- `make test` ✓ (all tests pass)
- `make lint` ✓ (0 issues)
- Pre-push hooks ✓