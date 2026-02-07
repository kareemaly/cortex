---
id: 088d24c6-0d8e-44ff-bd26-8fe418b147e3
author: claude
type: review_requested
created: 2026-02-07T08:13:08.546254Z
action:
    type: git_diff
    args:
        commit: 8bea60d
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Storage Rewrite: Frontmatter + Directory-per-Entity

### What changed

Rewrote the ticket and docs storage layers from JSON/flat-file to YAML frontmatter `index.md` files within directory-per-entity structures. Created a shared storage foundation and an independent session store.

### New packages

**`internal/storage/`** — Shared foundation (7 files):
- `errors.go` — Deduplicated `NotFoundError`, `ValidationError`, `IsNotFound`
- `slug.go` — `GenerateSlug(title, fallback)`, `ShortID()`, `DirName()`
- `frontmatter.go` — Generic `ParseFrontmatter[T]` / `SerializeFrontmatter[T]`
- `atomic.go` — `AtomicWriteFile` (temp file + rename)
- `comment.go` — Shared `Comment`/`CommentMeta`/`CommentAction` types, `CreateComment()`, `ListComments()`
- Full test coverage for all modules

**`internal/session/`** — Independent ephemeral session store (3 files):
- `Session` struct with flat `Status`/`Tool` fields (no history)
- JSON file-backed `Store` with file-level mutex
- CRUD: `Create`, `Get`, `GetByTicketID`, `UpdateStatus`, `End`, `List`
- Handles missing/empty file gracefully

### Rewritten packages

**`internal/ticket/`**:
- `Ticket` now uses `TicketMeta` with YAML tags (no more `Dates` struct, `Session` field)
- Store uses directory layout: `{status}/{slug}-{shortid}/index.md` + `comment-{shortid}.md`
- Title changes trigger `os.Rename` on entity directory
- `Get()` loads comments; `List()` does not (performance)
- Re-exports shared types as aliases for backward compat
- Removed: `SetSession`, `EndSession`, `UpdateSessionStatus` (moved to session store)

**`internal/docs/`**:
- `Doc` uses `DocMeta` — category derived from directory path, NOT in frontmatter
- Same directory-per-entity pattern as tickets
- **New capability**: `AddComment()` / `ListComments()` on docs
- Comments loaded on `Get()`, not on `List()`

### Test results

All 4 packages pass (86 tests total):
- `internal/storage` — 20 tests
- `internal/session` — 13 tests
- `internal/ticket` — 26 tests
- `internal/docs` — 35 tests (including new comment tests)

### Breaking changes

Consuming code (`daemon/api/`, `daemon/mcp/`, `core/spawn/`) references old struct shapes (`Dates`, `Session`, `AgentStatus`, etc.) and will need updates in a follow-up ticket. This is expected per the ticket's "breaking change is fine" constraint.