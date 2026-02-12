---
id: 66fd60ca-49be-4702-93b4-c582e891f276
title: Enrich architect prompt injection with tags, docs, and session continuity
type: work
tags:
    - architect-prompt
    - api
    - injection
created: 2026-02-09T14:11:37.877711Z
updated: 2026-02-09T14:40:41.029968Z
---
## Goal

Enhance the architect kickoff prompt with richer context so the architect reuses existing tag taxonomy, is aware of recent docs, and picks up context from the last concluded session.

## Requirements

### 1. New `GET /tags` API endpoint

Add a project-scoped endpoint that aggregates tags across all tickets and docs, returning them sorted by frequency (descending).

- Route: `GET /tags` (requires `X-Cortex-Project` header)
- Response: `{ "tags": [{ "name": "string", "count": int }] }`
- Aggregates from both ticket store and docs store
- Sorted by count descending

Add corresponding SDK client method `ListTags()` in `internal/cli/sdk/client.go`.

### 2. Extend `buildArchitectPrompt()` in `internal/core/spawn/spawn.go`

After fetching tickets, also:

- Call `client.ListTags()` → take top 20 → format as comma-separated list
- Call `client.ListDocs("", "", "")` → sort by created descending → take 20 → format as list with title, id, created date

### 3. Extend `ArchitectKickoffVars` in `internal/prompt/template.go`

Add two new fields:
- `TopTags string` — formatted top 20 tags
- `DocsList string` — formatted recent 20 docs

### 4. Update KICKOFF.md template

Update `internal/install/defaults/claude-code/prompts/architect/KICKOFF.md` to render the new sections:

```markdown
# Project: {{.ProjectName}}

**Current date**: {{.CurrentDate}}

# Tickets

{{.TicketList}}
{{if .DocsList}}
# Recent Docs

{{.DocsList}}
{{end}}
{{if .TopTags}}
# Tags

Reuse existing tags when creating tickets: {{.TopTags}}
{{end}}
```

Add a session continuity nudge. Something like (conditionally, only when docs exist):

> If there are recent session docs below, start by reading the most recent one with `readDoc` to pick up context from the last session.

### 5. Tests

- Unit test for the new `/tags` endpoint
- Unit test for the SDK `ListTags()` method
- Verify `buildArchitectPrompt` includes tags and docs when available (and gracefully omits them when empty)

## Key Files

| File | Change |
|------|--------|
| `internal/daemon/api/server.go` | Add `GET /tags` route |
| `internal/daemon/api/tags.go` | New handler (or add to existing file) |
| `internal/daemon/api/types.go` | Add response type if needed |
| `internal/cli/sdk/client.go` | Add `ListTags()` method |
| `internal/prompt/template.go` | Extend `ArchitectKickoffVars` |
| `internal/core/spawn/spawn.go` | Extend `buildArchitectPrompt()` |
| `internal/install/defaults/claude-code/prompts/architect/KICKOFF.md` | Add docs, tags, continuity sections |

## Notes

- Tags and docs sections should be omitted from the prompt when empty (use Go template conditionals)
- The `/tags` endpoint is useful beyond injection — TUI and MCP can use it later
- Keep the fallback inline format in `buildArchitectPrompt` working (it doesn't need the new fields)