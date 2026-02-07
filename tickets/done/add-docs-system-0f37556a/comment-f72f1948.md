---
id: f72f1948-e38d-446e-af48-97fe0c55b85a
author: claude
type: review_requested
created: 2026-02-06T13:49:36.953068Z
action:
    type: git_diff
    args:
        commit: bba8c85
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Added a complete documentation system with MCP tools for architect sessions and cross-linking between docs and tickets via a `references` field.

### New packages/files (8 new files):
- **`internal/docs/`** — Core package: `Doc` model with YAML frontmatter, `Store` with CRUD + Move + List operations, slug generation, error types, frontmatter parsing/serialization
- **`internal/daemon/api/docs.go`** — 6 HTTP handlers: GET/POST /docs, GET/PUT/DELETE /docs/{id}, POST /docs/{id}/move
- **`internal/daemon/api/docs_store_manager.go`** — Per-project lazy-init store manager (same pattern as ticket StoreManager)
- **`internal/daemon/mcp/tools_docs.go`** — 6 MCP tool handlers: createDoc, readDoc, updateDoc, deleteDoc, moveDoc, listDocs
- **`internal/docs/store_test.go`** — 31 unit tests covering CRUD, move, list with filters, frontmatter round-trip, slug generation, validation

### Modified files (21 files):
- **Project config**: Added `DocsConfig` with configurable path, merge support
- **Events**: Added `DocCreated`, `DocUpdated`, `DocDeleted` event types
- **Shared types**: Added `DocResponse`, `DocSummary`, `ListDocsResponse` + conversion functions
- **HTTP layer**: Added doc route group, request/response types, error handling, deps
- **SDK client**: Added 6 doc methods (CreateDoc, GetDoc, UpdateDoc, DeleteDoc, MoveDoc, ListDocs)
- **MCP types**: Added doc input/output types + ticket References fields
- **MCP tools_architect.go**: Registered 6 doc tools, pass references in ticket create/update
- **Ticket model**: Added `References []string` field to Ticket struct
- **Ticket store**: Updated Create/Update signatures to include references
- **Test files**: Updated all callers of Create/Update with new signatures (3 test files)

### Verification:
- `make build` — passes
- `make test` — all 14 packages pass (including new `internal/docs`)
- `make lint` — 0 issues