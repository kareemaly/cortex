---
id: f2e7fa87-029c-4540-97ea-58a9aad6feb5
author: claude
type: done
created: 2026-02-07T11:06:27.71092Z
---
Removed two migration scripts that had served their purpose:

- `cmd/migrate/main.go` (Go, JSON→YAML frontmatter migration)
- `ephemeral/migrate_tickets.py` (Python, cortex0 migration)

Build and all unit tests pass. Commit: d2d352d.