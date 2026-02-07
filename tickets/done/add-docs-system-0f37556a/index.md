---
id: 0f37556a-a24a-46fb-b4ec-a2bc621a806b
title: Add docs system with MCP tools for architect
type: work
created: 2026-02-06T13:14:51.241314Z
updated: 2026-02-06T13:55:25.736522Z
---
## Summary

Add a documentation system (like Confluence) to Cortex — persistent, human-readable markdown files that architects can create, search, categorize, and link to tickets. Docs are distinct from tickets: they have no status lifecycle, are long-lived, and are meant to be committed to git.

## Storage

- Plain `.md` files with YAML frontmatter
- Configurable location in `.cortex/cortex.yaml`:
  ```yaml
  docs:
    path: docs/    # relative to project root, default: .cortex/docs/
  ```
- Subdirectories as categories: `specs/`, `findings/`, `decisions/`, etc. (free-form, created on demand)
- Each doc has a stable UUID in frontmatter — all references use IDs, never paths
- Filename format: `{slug}-{shortID}.md` (same pattern as tickets)

### Frontmatter schema

```yaml
---
id: a8751b1b-...
title: Authentication API
category: api
tags: [auth, security]
references: ["ticket:0428bc92", "doc:b3f2c1a9"]
created: 2026-02-06T12:00:00Z
updated: 2026-02-06T12:00:00Z
---
```

## MCP Tools (architect only)

### `createDoc`
| Param | Required | Type | Notes |
|-------|----------|------|-------|
| `title` | ✅ | string | Generates slug for filename |
| `category` | ✅ | string | Subdirectory name |
| `body` | ❌ | string | Markdown content |
| `tags` | ❌ | string[] | Free-form tags |
| `references` | ❌ | string[] | e.g. `["ticket:abc123"]` |
| `project_path` | ❌ | string | Cross-project |

### `readDoc`
| Param | Required | Type | Notes |
|-------|----------|------|-------|
| `id` | ✅ | string | |
| `project_path` | ❌ | string | |

### `updateDoc`
| Param | Required | Type | Notes |
|-------|----------|------|-------|
| `id` | ✅ | string | |
| `title` | ❌ | string | Re-slugs filename |
| `body` | ❌ | string | Full replacement |
| `tags` | ❌ | string[] | Full replacement |
| `references` | ❌ | string[] | Full replacement |
| `project_path` | ❌ | string | |

### `deleteDoc`
| Param | Required | Type | Notes |
|-------|----------|------|-------|
| `id` | ✅ | string | Current project only, no `project_path` |

### `moveDoc`
| Param | Required | Type | Notes |
|-------|----------|------|-------|
| `id` | ✅ | string | |
| `category` | ✅ | string | Target subdirectory |
| `project_path` | ❌ | string | |

### `listDocs`
| Param | Required | Type | Notes |
|-------|----------|------|-------|
| `category` | ❌ | string | Filter by subdirectory |
| `tag` | ❌ | string | Filter by tag |
| `query` | ❌ | string | Search title + body content |
| `project_path` | ❌ | string | |

Returns summaries: id, title, category, tags, created/updated dates, and matching snippets when query is used.

## Ticket references extension

Extend existing `createTicket` and `updateTicket` MCP tools to accept an optional `references` field (`string[]`). Same format: `["doc:a8751b1b"]`. Stored in ticket JSON. Surfaced on `readTicket`.

## Implementation notes

- New store: `internal/docs/store.go` — read/write/search markdown files with frontmatter parsing
- New MCP tools: `internal/daemon/mcp/tools_architect.go` — add 6 doc tools to architect tool set
- New API endpoints in `internal/daemon/api/` for HTTP access
- Config: add `docs.path` to project config schema in `internal/project/config/config.go`
- Daemon resolves doc ID → path by scanning files (no index)
- StoreManager pattern: one docs store per project, lazy init
- Re-slug filename on title update (safe because references use IDs)
- Categories are free-form subdirectories, created on demand