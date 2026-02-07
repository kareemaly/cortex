---
id: 5a8372df-3b72-401f-a7ba-34b51d77595c
author: claude
type: done
created: 2026-02-06T13:55:25.73296Z
---
## Summary

Implemented a complete documentation system for Cortex with MCP tools and ticket cross-references.

### What was done:

**New packages/files (9 new files):**
- `internal/docs/` — Core package with Doc model (YAML frontmatter + markdown body), Store (CRUD + Move + List with category/tag/query filters), slug generation, error types, frontmatter parsing/serialization
- `internal/docs/store_test.go` — 31 unit tests covering all store operations
- `internal/daemon/api/docs.go` — 6 HTTP handlers (GET/POST /docs, GET/PUT/DELETE /docs/{id}, POST /docs/{id}/move)
- `internal/daemon/api/docs_store_manager.go` — Per-project lazy-init store manager
- `internal/daemon/mcp/tools_docs.go` — 6 MCP tool handlers for architect sessions

**Modified files (23 files):**
- Project config: Added DocsConfig with configurable path and merge support
- Events: Added DocCreated, DocUpdated, DocDeleted event types
- Shared types: Added DocResponse, DocSummary, ListDocsResponse + conversion functions
- HTTP layer: Added /docs route group, request/response types, error handling, deps
- SDK client: Added 6 doc methods
- MCP layer: Added doc input/output types, registered 6 doc tools in architect
- Ticket model: Added References field for cross-linking between docs and tickets
- Ticket store: Updated Create/Update signatures to include references
- All test files updated for new signatures (3 test files)
- CLAUDE.md and README.md updated with docs system documentation

**Commits:**
- `bba8c85` — feat: add docs system with MCP tools and ticket references
- `f2c0991` — docs: update CLAUDE.md and README.md for docs system

**Verification:** make build passes, make test (all 14 packages pass including new internal/docs), make lint (0 issues), pre-push hooks passed.