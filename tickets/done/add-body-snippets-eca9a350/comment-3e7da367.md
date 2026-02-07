---
id: 3e7da367-32aa-488e-8905-78a3185c937c
author: claude
type: review_requested
created: 2026-02-06T14:12:30.987698Z
action:
    type: git_diff
    args:
        commit: b592b4e
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Added body snippet support to `listDocs` query results. When a `query` parameter is provided, each `DocSummary` now includes a `snippet` field with ~150 characters of body content centered on the first case-insensitive match.

## Changes

1. **`internal/types/response.go`** — Added `Snippet string \`json:"snippet,omitempty"\`` to `DocSummary` struct
2. **`internal/types/convert.go`** — Added `ExtractSnippet(body, query, maxLen)` helper and `ToDocSummaryWithQuery(doc, query)` converter
3. **`internal/daemon/api/docs.go`** — Updated `List` handler to use `ToDocSummaryWithQuery` when query is non-empty
4. **`internal/daemon/mcp/types.go`** — Added `Snippet` field to `DocSummaryOutput` and passed it through in `docSummaryToOutput`
5. **`internal/types/convert_test.go`** — 11 tests covering: empty query, empty body, no match, short body, match at start/end/middle, case-insensitive match, and `ToDocSummaryWithQuery` integration

## Verification

- `make build` — clean
- `make lint` — 0 issues
- `make test` — all tests pass (11 new tests in `internal/types/`)