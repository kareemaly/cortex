---
id: 3bc26c74-12a8-40f2-9598-8fa9d2c67600
author: claude
type: review_requested
created: 2026-02-07T10:50:33.507714Z
action:
    type: git_diff
    args:
        commit: 75a9ab1
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Migration Script: JSON → YAML Frontmatter

Added `cmd/migrate/main.go` — a standalone Go program that migrates old-format JSON tickets to the new YAML frontmatter + directory-per-entity format.

### What it does
- Reads `.cortex/tickets/{status}/{slug}-{shortid}.json` files
- Writes `tickets/{status}/{slug}-{shortid}/index.md` with YAML frontmatter
- Writes `comment-{shortid}.md` files inside each entity directory
- Converts action args from JSON to YAML-compatible maps

### Already run on ~/kesc
- **13 tickets** migrated (2 backlog, 11 done)
- **12 comments** migrated (including 3 with git_diff actions)
- Output verified: frontmatter fields, due dates, comment types, and action args all correct

### Reuses existing packages
- `internal/storage` — `DirName()`, `ShortID()`, `SerializeFrontmatter()`, `AtomicWriteFile()`
- `internal/ticket` — `TicketMeta` struct
- No new dependencies added