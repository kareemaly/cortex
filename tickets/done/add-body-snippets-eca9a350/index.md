---
id: eca9a350-bbe8-47d8-b93e-e69020c7a3ed
title: Add body snippets to listDocs query results
type: work
created: 2026-02-06T14:04:00.387683Z
updated: 2026-02-06T14:13:06.645426Z
---
## Summary

When `listDocs` is called with a `query` parameter, the results should include a snippet of the matching body content around the match — not just the standard summary (id, title, category, tags, dates).

This improves discoverability by letting the architect see *why* a doc matched without needing to `readDoc` each result.

## Current behavior

`listDocs` with `query` returns `DocSummary` (id, title, category, tags, created, updated) — no body context.

## Expected behavior

When `query` is provided, each result should include a `snippet` field with a short excerpt (~150 chars) of the body content surrounding the first match, with the match highlighted or clearly visible.

## Scope

- Add `Snippet` field to `DocSummary` response type (`internal/types/response.go`)
- Populate snippet in the store's `List()` method when query matches body content (`internal/docs/store.go`)
- Update conversion function (`internal/types/convert.go`)
- Update MCP output type (`internal/daemon/mcp/types.go`)

## References

Follow-up from `doc:0f37556a` (Add docs system with MCP tools for architect).