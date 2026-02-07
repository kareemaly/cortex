---
id: b67f99fc-7baf-4c38-b793-d7d85877530c
author: claude
type: done
created: 2026-02-06T14:13:06.642567Z
---
## Summary

Added body snippet support to `listDocs` query results.

### What was done
When `listDocs` is called with a `query` parameter, each result now includes a `snippet` field with ~150 characters of body content centered on the first case-insensitive match. This lets the architect see *why* a doc matched without needing to `readDoc` each result individually.

### Files changed
- `internal/types/response.go` — Added `Snippet` field to `DocSummary`
- `internal/types/convert.go` — Added `ExtractSnippet` helper and `ToDocSummaryWithQuery` converter
- `internal/daemon/api/docs.go` — Updated List handler to populate snippets when query is provided
- `internal/daemon/mcp/types.go` — Added `Snippet` to `DocSummaryOutput` and passthrough in conversion
- `internal/types/convert_test.go` — 11 unit tests covering edge cases

### Verification
- Build: clean
- Lint: 0 issues
- Tests: all pass (11 new)
- Pushed to origin/main (commit b592b4e)