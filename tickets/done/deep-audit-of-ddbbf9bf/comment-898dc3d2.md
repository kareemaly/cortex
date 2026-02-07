---
id: 898dc3d2-b6ba-4651-bb71-e640ebb6b27f
author: claude
type: comment
created: 2026-02-07T10:28:44.172213Z
---
## Audit Finding: CLEAN — Ticket Store (internal/ticket/)

**Files:** ticket.go (63 lines), store.go (434 lines), store_test.go (605 lines)

### Verified:
- **Filesystem structure**: `tickets/{status}/{slug}-{shortid}/index.md` — confirmed by TestStoreDirLayout
- **Frontmatter round-trip**: TicketMeta with yaml tags (not json). All fields survive serialization: ID, Title, Type, Tags, References, Due, Created, Updated
- **Comments**: Separate `comment-{shortid}.md` files in entity directory. Loaded on Get(), skipped on List() for performance.
- **Title rename**: Update() detects title change, renames entity directory atomically via os.Rename
- **Tags/references/due date**: All persisted with yaml omitempty. SetDueDate/ClearDueDate properly handle nil.
- **Concurrency**: Per-entity mutex via sync.Map. All 6 mutating operations (Create, Update, SetDueDate, Delete, Move, AddComment) lock correctly with defer Unlock.
- **Uses storage package**: DirName, ShortID, SerializeFrontmatter, ParseFrontmatter, CreateComment, ListComments, AtomicWriteFile, IsNotFound
- **Old files deleted**: slug.go, slug_test.go, ticket_test.go, errors.go — all moved to storage/
- **Event emission**: TicketCreated, TicketUpdated, TicketMoved, TicketDeleted, CommentAdded — all emitted correctly
- **All 32 ticket tests PASS** including 2 concurrency tests (10 goroutines x 10 updates, 10 concurrent comment adds)

**Severity: CLEAN**