---
id: 25af1c8e-6f07-409e-bcfe-72294038d402
author: claude
type: review_requested
created: 2026-02-09T14:38:18.785389Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/8cce625d-d968-4e81-a30a-abe1e68fda76
        commit: 93f3238
---
## Changes

### New `GET /tags` endpoint (`internal/daemon/api/tags.go`)
- Aggregates tags from both ticket store and docs store
- Normalizes tags to lowercase (e.g., "API" and "api" merge)
- Sorts by count descending, alphabetical tiebreak
- Gracefully handles nil store managers
- Response: `{ "tags": [{ "name": "string", "count": int }] }`

### Response types (`internal/types/response.go`)
- Added `TagCount` and `ListTagsResponse` types
- Re-exported in `internal/daemon/api/types.go` and `internal/cli/sdk/client.go`

### SDK client (`internal/cli/sdk/client.go`)
- Added `ListTags()` method following standard pattern

### Route registration (`internal/daemon/api/server.go`)
- Registered `GET /tags` in project-scoped route group

### Template vars (`internal/prompt/template.go`)
- Added `TopTags` and `DocsList` fields to `ArchitectKickoffVars`

### Prompt building (`internal/core/spawn/spawn.go`)
- `buildArchitectPrompt()` now fetches top 20 tags (comma-separated) and recent 20 docs (sorted by created descending)
- Both fetches degrade gracefully on error (empty string → template conditional omits sections)

### KICKOFF.md templates (both claude-code and copilot)
- Added conditional `# Recent Docs` section with continuity nudge
- Added conditional `# Tags` section encouraging tag reuse
- Sections omitted when data is empty

### Tests (8 new tests, all passing)
- `internal/daemon/api/tags_test.go`: empty stores, ticket-only tags, merged ticket+doc tags, nil docs store manager
- `internal/cli/sdk/client_test.go`: `TestListTags_Success`, `TestListTags_Error`
- `internal/prompt/template_test.go`: template rendering with populated vs empty tags/docs

### Verification
- `make lint` — 0 issues
- `make test` — all pass
- `make build` — builds successfully