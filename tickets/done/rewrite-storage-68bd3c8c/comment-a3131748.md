---
id: a3131748-15e7-411b-ac72-77c10c415318
author: claude
type: done
created: 2026-02-07T08:13:20.760123Z
---
## Summary

Rewrote the storage layer from JSON/flat-file to YAML frontmatter + directory-per-entity.

### Files created
- `internal/storage/errors.go` — Shared NotFoundError, ValidationError, IsNotFound
- `internal/storage/slug.go` — GenerateSlug with fallback param, ShortID, DirName
- `internal/storage/slug_test.go` — 12 slug tests + ShortID/DirName tests
- `internal/storage/frontmatter.go` — Generic ParseFrontmatter[T] / SerializeFrontmatter[T]
- `internal/storage/frontmatter_test.go` — Round-trip, empty body, missing delimiter, invalid YAML tests
- `internal/storage/atomic.go` — AtomicWriteFile helper
- `internal/storage/comment.go` — Comment types, CreateComment, ListComments
- `internal/storage/comment_test.go` — Create, list, sort, action round-trip tests
- `internal/session/session.go` — Session and AgentStatus types
- `internal/session/store.go` — JSON file-backed ephemeral session store
- `internal/session/store_test.go` — Full CRUD + concurrent access tests

### Files rewritten
- `internal/ticket/ticket.go` — TicketMeta with YAML tags, re-exported shared types
- `internal/ticket/store.go` — Directory-per-entity store with index.md + comment files
- `internal/ticket/store_test.go` — 26 tests covering all operations
- `internal/docs/doc.go` — DocMeta with YAML tags, category from path
- `internal/docs/store.go` — Directory-per-entity store with comment support
- `internal/docs/store_test.go` — 35 tests including new comment + category-from-path tests

### Files deleted
- `internal/ticket/errors.go` → moved to storage/
- `internal/ticket/slug.go` → moved to storage/
- `internal/ticket/slug_test.go` → moved to storage/
- `internal/ticket/ticket_test.go` → tests integrated into store_test.go
- `internal/docs/errors.go` → moved to storage/
- `internal/docs/frontmatter.go` → moved to storage/
- `internal/docs/slug.go` → moved to storage/

### Test results
All 86 tests pass across 4 packages. go vet clean.