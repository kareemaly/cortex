---
id: 99e2a90b-0744-4a83-a267-014e977e519c
author: claude
type: review_requested
created: 2026-02-15T08:50:12.264068Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/977f87a3-08ab-4c8a-b8c2-596cfafc3921
        commit: 6d65d29
---
## Root Cause

Two issues:

1. Commit 847e3da removed `cortex show` but left 4 stale references in daemon/spawn/SDK code
2. The show subcommand had `Use: "<id>"` — Cobra treats angle brackets as a literal command name, not a placeholder, so routing never worked

## Fix (5 files, single commit)

- **`cmd/cortex/commands/ticket_show.go`** — `Use: "<id>"` → `Use: "show <id>"` so Cobra routes correctly
- **`cmd/cortex/commands/ticket.go`** — Kept as simple group command (reverted positional arg delegation)
- **`internal/daemon/api/tickets.go`** — Popup command: `cortex ticket show %s`
- **`internal/core/spawn/spawn.go`** (2 locations) — Companion cmd: `cortex ticket show %s` (removed redundant `CORTEX_TICKET_ID` env var prefix since ID is now an explicit arg)
- **`internal/cli/sdk/client.go`** — Updated comment

## Verification

- `make build` — compiles
- `make lint` — 0 issues  
- `make test` — all pass
- Manual: `make install` from worktree, then `cortex ticket show <id>`