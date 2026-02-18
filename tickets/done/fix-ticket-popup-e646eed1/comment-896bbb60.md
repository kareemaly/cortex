---
id: 896bbb60-def0-46cc-b154-85a90deeb11e
author: claude
type: comment
created: 2026-02-15T08:42:03.102494Z
---
## Root Cause Analysis

The `cortex show` CLI command was deleted in commit `847e3da` ("remove dead CLI commands") but **4 references** in daemon/spawn code were not updated. The replacement is `cortex ticket <id>`.

### Affected locations (all `.go` files referencing `cortex show`):

1. **`internal/daemon/api/tickets.go:1040`** — `ShowPopup` handler builds command `cortex show %s` for the kanban popup. **This is the direct cause of the reported bug.**

2. **`internal/core/spawn/spawn.go:553`** — Companion pane command `CORTEX_TICKET_ID=%s cortex show` in the first spawn path (non-worktree). Broken because `cortex ticket` requires an explicit ID argument (`ExactArgs(1)`), while the old `show` accepted `MaximumNArgs(1)` with env var fallback.

3. **`internal/core/spawn/spawn.go:1012`** — Same companion pane command in the `spawnInTmux` function (worktree path). Same issue.

4. **`internal/cli/sdk/client.go:959`** — Comment-only reference ("opens cortex show for a ticket").

### Key difference between old and new commands:
- Old `cortex show [id]` — `MaximumNArgs(1)`, could read `CORTEX_TICKET_ID` env var as fallback
- New `cortex ticket <id>` — `ExactArgs(1)`, requires explicit ID argument

This means the spawn.go companion commands need to pass the ticket ID as an argument (changing the format string from one `%s` to two `%s` args).