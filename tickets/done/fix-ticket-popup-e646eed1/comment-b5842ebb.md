---
id: b5842ebb-7454-47cd-b07f-8b82f14c7b7d
author: claude
type: review_requested
created: 2026-02-15T08:43:43.229783Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/977f87a3-08ab-4c8a-b8c2-596cfafc3921
        commit: bfdb466
---
## Root Cause

Commit 847e3da removed the `cortex show` CLI command but left 4 stale references across 3 files. The ticket popup and spawn companion panes were invoking a nonexistent command.

## Fix

Updated all references from `cortex show` to `cortex ticket`:

1. **`internal/daemon/api/tickets.go`** — Updated comment (line 1015) and popup command string (line 1040) from `cortex show %s` to `cortex ticket %s`
2. **`internal/core/spawn/spawn.go`** — Two companion command strings (lines 553, 1012): changed from `"CORTEX_TICKET_ID=%s cortex show"` to `"CORTEX_TICKET_ID=%s cortex ticket %s"` with `req.TicketID` passed twice in Sprintf (env var kept for backwards compat, explicit arg added for the new command's `ExactArgs(1)` requirement)
3. **`internal/cli/sdk/client.go`** — Updated comment (line 959)

## Verification

- `make build` — compiles cleanly
- `make lint` — 0 issues
- `make test` — all unit tests pass