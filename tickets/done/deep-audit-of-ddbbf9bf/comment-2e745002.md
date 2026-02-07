---
id: 2e745002-12b8-4ec4-a832-db5ded95aec8
author: claude
type: comment
created: 2026-02-07T10:28:51.292496Z
---
## Audit Finding: CLEAN — Doc Store (internal/docs/)

**Files:** doc.go, store.go, store_test.go

### Verified:
- **Filesystem structure**: `docs/{category}/{slug}-{shortid}/index.md`
- **Category from directory**: TestCategoryNotInFrontmatter confirms category is derived from parent directory, NOT stored in YAML frontmatter
- **Comment support (new)**: Full flow — AddComment() → storage.CreateComment() → writes comment-{shortid}.md. Get() loads comments. List() skips comments for performance.
- **Orphaned files removed**: errors.go and frontmatter.go successfully deleted (moved to storage/)
- **Concurrency**: Per-doc mutex via sync.Map, same pattern as ticket store
- **Filters**: Case-insensitive tag and query filtering on List()
- **All 40 doc tests PASS**

**Severity: CLEAN**