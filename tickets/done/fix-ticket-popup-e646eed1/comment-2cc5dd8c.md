---
id: 2cc5dd8c-ed62-4141-b4d2-61a3aa175101
author: claude
type: comment
created: 2026-02-15T08:43:26.06261Z
---
Root cause: Commit 847e3da removed the `cortex show` CLI command but left 4 references to it across 3 files:
- `internal/daemon/api/tickets.go`: comment and `fmt.Sprintf("cortex show %s", ticketID)` for popup command
- `internal/core/spawn/spawn.go`: two companion command strings using `"CORTEX_TICKET_ID=%s cortex show"` (env var fallback pattern)
- `internal/cli/sdk/client.go`: comment referencing `cortex show`

The replacement command `cortex ticket <id>` requires an explicit ID argument (ExactArgs(1)), so the spawn companion commands also needed the ticket ID passed as an argument (not just via env var).