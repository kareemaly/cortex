---
id: 24779da6-6084-4c2c-827f-f0e752f7a33c
author: claude
type: done
created: 2026-02-09T14:40:41.028971Z
---
## Summary

Enriched the architect kickoff prompt with tags and docs context so architects start sessions with awareness of the project's tag taxonomy and recent documentation.

### Changes Made

**New `GET /tags` endpoint** (`internal/daemon/api/tags.go`)
- Aggregates tags from both ticket and docs stores with lowercase normalization
- Returns sorted by count descending (alphabetical tiebreak)
- Gracefully handles nil store managers

**Response types** (`internal/types/response.go`)
- Added `TagCount` and `ListTagsResponse` types
- Re-exported in `api/types.go` and `sdk/client.go`

**SDK client** (`internal/cli/sdk/client.go`)
- Added `ListTags()` method

**Route registration** (`internal/daemon/api/server.go`)
- Registered `GET /tags` in project-scoped route group

**Template vars** (`internal/prompt/template.go`)
- Added `TopTags` and `DocsList` fields to `ArchitectKickoffVars`

**Prompt building** (`internal/core/spawn/spawn.go`)
- `buildArchitectPrompt()` fetches top 20 tags and recent 20 docs
- Both degrade gracefully on error

**KICKOFF.md templates** (both claude-code and copilot)
- Conditional `# Recent Docs` section with continuity nudge
- Conditional `# Tags` section encouraging tag reuse

**Tests** (8 new tests, all passing)
- 4 handler tests: empty stores, ticket-only, merged, nil docs manager
- 2 SDK tests: success and error cases
- 2 template tests: populated vs empty vars

**Documentation** — Updated CLAUDE.md API Endpoints section

### Files Changed
- `internal/types/response.go` (modified)
- `internal/daemon/api/types.go` (modified)
- `internal/daemon/api/tags.go` (new)
- `internal/daemon/api/tags_test.go` (new)
- `internal/daemon/api/server.go` (modified)
- `internal/cli/sdk/client.go` (modified)
- `internal/cli/sdk/client_test.go` (modified)
- `internal/prompt/template.go` (modified)
- `internal/prompt/template_test.go` (new)
- `internal/core/spawn/spawn.go` (modified)
- `internal/install/defaults/claude-code/prompts/architect/KICKOFF.md` (modified)
- `internal/install/defaults/copilot/prompts/architect/KICKOFF.md` (modified)
- `CLAUDE.md` (modified)