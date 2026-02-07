---
id: ddbbf9bf-4dd4-4f73-9868-e0602edd0bb9
title: Deep audit of storage migration before merge to main
type: research
created: 2026-02-07T10:23:26.262508Z
updated: 2026-02-07T10:32:51.959299Z
---
## Overview

Comprehensive audit of ALL changes on the `feat/frontmatter-storage` branch vs `main`. This is the final gate before merging and releasing v1.0.0. The goal is to find any issues — dead code, broken data flows, missing functionality, stale references, incomplete migrations, tech debt, or anything that would make this not production-ready.

**This is a read-only research ticket. Do NOT make code changes. Report ALL findings as comments.**

## Audit Scope

### 1. Full diff analysis
- Run `git diff main...HEAD --stat` to see every file changed
- Review each changed file for correctness and completeness
- Look for: TODOs, FIXMEs, commented-out code, debug prints, placeholder implementations

### 2. Data flow verification

Trace these complete data flows end-to-end:

**Ticket lifecycle:**
- Create ticket (MCP `createTicket` → SDK → HTTP API → ticket store → filesystem)
- Read ticket (MCP `readTicket` → SDK → HTTP API → ticket store → parse frontmatter → response with comments)
- Update ticket (including tags, due date, title rename → directory rename)
- Move ticket (status change → directory move)
- Delete ticket (remove entire entity directory)
- Add comment (MCP → SDK → HTTP API → store → creates `comment-{shortid}.md`)

**Doc lifecycle:**
- Same CRUD flows as tickets but with category instead of status
- Doc comments (new capability — verify full flow)

**Session lifecycle:**
- Spawn session → session store create → tmux window
- Update session status (agent status reporting)
- End session → session store delete (ephemeral)
- List sessions (MCP `listSessions` → SDK → HTTP API → session store → resolve ticket titles)

**CLI/TUI flows:**
- `cortex init` → creates correct directory structure at project root
- `cortex kanban` → fetches tickets via SDK → renders with tags, type, due, session status
- `cortex show <id>` → fetches ticket detail → renders flat dates, tags, comments with author, session section
- `cortex ticket list` → renders with TYPE column
- `cortex ticket spawn <id>` → spawn flow with session store

### 3. API completeness

Verify all HTTP endpoints in `server.go` are:
- Properly routed
- Have corresponding SDK client methods
- Have corresponding MCP tools (where applicable)
- Handle errors correctly

Check specifically:
- `GET /sessions/` — new endpoint
- `POST /docs/{id}/comments` — new endpoint
- All existing endpoints work with new types

### 4. MCP tool completeness

Verify all 20 architect tools and 5 ticket agent tools:
- Tool registration matches handler
- Input types have correct JSON schema annotations for MCP
- Output types return all expected fields
- New tools: `addDocComment`, `listSessions` — verify full implementation

### 5. Store layer integrity

**Ticket store (`internal/ticket/`):**
- All CRUD operations create correct filesystem structure
- Frontmatter serialization/deserialization round-trips correctly
- Comments stored as separate files, sorted by timestamp
- Title rename triggers directory rename
- Tags, references, due date properly persisted
- Concurrency: per-entity mutexes work correctly

**Doc store (`internal/docs/`):**
- Same checks as ticket store
- Category derived from directory path (not in frontmatter)
- Comment support works end-to-end

**Session store (`internal/session/`):**
- Ephemeral: entries deleted on End()
- JSON file at `.cortex/sessions.json`
- Handles missing/empty file gracefully
- GetByTicketID works correctly

**Shared storage (`internal/storage/`):**
- Frontmatter parsing handles edge cases (empty body, no frontmatter, invalid YAML)
- Slug generation correct (max 20 chars, word boundary truncation)
- Atomic writes working (temp file + rename)
- Comment creation/listing correct

### 6. Config verification

- `internal/project/config/config.go`: `TicketsPath()` defaults to `{root}/tickets`, `DocsPath()` defaults to `{root}/docs`
- Custom paths (relative and absolute) work correctly
- `cortex init` respects config

### 7. Spawn orchestration

- 3 states: normal, active, orphaned (no more "ended")
- 3 modes: normal, resume, fresh
- State × mode matrix fully handled
- Session store properly used for all session operations
- `--resume` flag generation correct (bare vs with ID)

### 8. Notification dispatcher

- Works with new session model
- Correctly looks up sessions from session store
- Event types match new storage events

### 9. Stale reference scan

Search the ENTIRE codebase for any remaining references to old patterns:
- `\.Dates\.` or `Dates{` — old nested dates
- `ticket\.Session` — old embedded session (outside ticket package)
- `ticket\.AgentStatus` — moved to session
- `ticket\.StatusEntry` — moved to session
- `SessionID` on comment types — should be `Author`
- `.cortex/tickets` or `.cortex/docs` hardcoded (should use config)
- `StateEnded` — removed state
- `json:"` tags on ticket/doc structs that should be `yaml:"`
- Any `encoding/json` imports in ticket/doc store (should use yaml)

### 10. Test coverage

- Are there tests for all new tools (addDocComment, listSessions)?
- Are there tests for tag filtering on listTickets?
- Do integration tests cover the new paths?
- Any test files still referencing old types?

## Output

Add ALL findings as ticket comments. Structure as:

1. **Critical** — blocks merge (broken functionality, data loss risk)
2. **Important** — should fix before merge (missing tests, incomplete flows)
3. **Minor** — nice to fix (naming, comments, minor cleanup)
4. **Clean** — areas verified as correct (for confidence)

If there are critical or important findings, call `requestReview` with a summary. If everything is clean, call `requestReview` confirming merge readiness.

## Branch

Auditing `feat/frontmatter-storage` branch vs `main`.