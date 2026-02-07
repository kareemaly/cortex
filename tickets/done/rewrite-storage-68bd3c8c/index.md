---
id: 68bd3c8c-afde-408c-80bd-f386b9ee4387
title: Rewrite storage packages to frontmatter + directory-per-entity
type: work
created: 2026-02-07T07:50:48.634053Z
updated: 2026-02-07T08:13:20.762167Z
---
## Overview

Rewrite the ticket and doc storage layers from JSON/single-file to YAML frontmatter + directory-per-entity. Create a new independent session store. This is the foundation for the full storage migration — no daemon/API/TUI changes in this ticket.

## Design

### New Directory Layout

```
tickets/                          # default at project root (configurable)
├── backlog/
│   └── fix-auth-bug-a1b2c3d4/
│       ├── index.md
│       └── comment-f7a8b9c0.md
├── progress/
├── review/
└── done/

docs/                             # default at project root (configurable)
├── specs/
│   └── api-design-d1e2f3a4/
│       └── index.md
└── decisions/

.cortex/sessions.json             # active sessions only, ephemeral
```

### Ticket `index.md` Frontmatter

```yaml
---
id: a1b2c3d4-e5f6-7890-abcd-ef0123456789
title: Fix the authentication bug
type: work
tags:
  - auth
  - critical
references:
  - doc:d1e2f3a4-e5f6-7890-abcd-ef0123456789
due: 2026-03-15
created: 2026-02-07T10:00:00Z
updated: 2026-02-07T14:30:00Z
---
Body markdown here
```

- `type` values: `work`, `debug`, `research`, `chore`
- Status is encoded in the **directory path** (backlog/, progress/, review/, done/) — NOT in frontmatter
- No lifecycle date fields (progress_at, reviewed_at, done_at) — derive from comment timestamps
- Folder naming: `{slug}-{shortid}/` where shortid = first 8 chars of UUID, slug max 20 chars (existing slug logic)

### Doc `index.md` Frontmatter

```yaml
---
id: d1e2f3a4-e5f6-7890-abcd-ef0123456789
title: API Design Specification
tags:
  - api
  - v2
references:
  - ticket:a1b2c3d4-e5f6-7890-abcd-ef0123456789
created: 2026-02-07T10:00:00Z
updated: 2026-02-07T14:30:00Z
---
Doc body here
```

- Category is in the **directory path** (specs/, decisions/, etc.) — NOT in frontmatter
- Add comment support to docs (new capability — same pattern as tickets)

### Comment `comment-{shortid}.md` Frontmatter

```yaml
---
id: b5c6d7e8-e5f6-7890-abcd-ef0123456789
author: claude
type: comment
created: 2026-02-07T15:00:00Z
action:
  type: git_diff
  args:
    repo_path: /path/to/repo
    commit: abc123
---
Comment body markdown here
```

- Comment types: `comment`, `review_requested`, `done`, `blocker`
- `author` replaces `session_id` — can be agent name or "human"
- `action` field preserved for structured actions (git_diff, etc.)
- Same format for both ticket comments and doc comments

### Session Store `.cortex/sessions.json`

```json
{
  "a1b2c3d4": {
    "ticket_id": "a1b2c3d4-e5f6-7890-abcd-ef0123456789",
    "agent": "claude",
    "tmux_window": "fix-auth-bug",
    "worktree_path": "/path/to/worktree",
    "feature_branch": "ticket/fix-auth-bug",
    "started_at": "2026-02-07T15:00:00Z",
    "status": "in_progress",
    "tool": "Edit"
  }
}
```

- Keyed by ticket short ID for fast lookup
- Ephemeral — entries deleted when session ends
- Entire file is disposable runtime state

## Implementation Scope

### 1. Rewrite `internal/ticket/`

- New `Ticket` struct with YAML tags instead of JSON
- `Comment` struct with YAML tags (id, author, type, created, action)
- Remove `Session` struct from ticket package (moves to session package)
- Remove `Dates` struct — replace with simple `created`, `updated`, `due` fields on Ticket
- Remove `StatusEntry`, `AgentStatus` types (move to session package)
- Rewrite `Store` — directory-per-entity operations:
  - `Create()` — create `{status}/{slug}-{shortid}/index.md`
  - `Get()` — scan status dirs, find entity dir, parse `index.md`
  - `Update()` — update frontmatter fields, handle slug rename (directory rename)
  - `Delete()` — remove entire entity directory
  - `List()` — scan status dir, parse all `index.md` files
  - `Move()` — move entity directory between status dirs
  - `AddComment()` — create `comment-{shortid}.md` in entity dir
  - `ListComments()` — scan entity dir for comment files, sort by created timestamp
- Update frontmatter parsing/serialization (reuse from docs or create shared util)
- Remove all session-related methods from ticket store
- Comprehensive unit tests for all operations including concurrency

### 2. Rewrite `internal/docs/`

- Update `Doc` struct — remove `category` field (derived from path), keep everything else
- Add `Comment` struct (same as ticket comments, or use shared type)
- Rewrite `Store` — directory-per-entity operations:
  - Same pattern as ticket store but with category dirs instead of status dirs
  - Add `AddComment()`, `ListComments()` methods (new capability)
- Update frontmatter serialization
- Comprehensive unit tests

### 3. Create `internal/session/`

- New package for independent session management
- `Session` struct with: id, ticket_id, agent, tmux_window, worktree_path, feature_branch, started_at, status, tool
- `AgentStatus` type: starting, in_progress, idle, waiting_permission, error
- `Store` backed by single JSON file (`.cortex/sessions.json`)
  - `Create()` — add session keyed by ticket short ID
  - `Get()` — lookup by ticket short ID
  - `Update()` — update status/tool fields
  - `End()` — remove entry (ephemeral)
  - `List()` — return all active sessions
- File-level mutex (single file, not per-entity)
- Atomic writes
- Unit tests

### 4. Shared utilities (if needed)

- Consider shared frontmatter parse/serialize if ticket and doc implementations converge
- Shared comment type if identical between tickets and docs
- Slug generation already exists — may just need minor updates

## Key Constraints

- **Breaking change is fine** — single user, full rewrite
- **No tech debt** — clean implementation, no compatibility layers
- **No daemon/API/MCP/TUI changes** — those are separate tickets
- **Existing tests for the old store can be adapted** — same behavioral contract, new format
- **Concurrency model stays the same** — per-entity mutexes, atomic writes, event bus emission

## Branch

Working on `feat/frontmatter-storage` branch.