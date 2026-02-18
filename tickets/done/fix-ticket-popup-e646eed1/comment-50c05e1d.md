---
id: 50c05e1d-d8bb-4f5a-a3f4-5572e5d36481
author: claude
type: done
created: 2026-02-15T08:51:28.914259Z
---
## Root Cause

Two issues caused the ticket popup/companion pane to fail:

1. **Stale references**: Commit 847e3da removed the `cortex show` CLI command but left 4 references to it across `internal/daemon/api/tickets.go`, `internal/core/spawn/spawn.go` (2 locations), and `internal/cli/sdk/client.go`
2. **Cobra routing bug**: The replacement subcommand `ticketShowCmd` had `Use: "<id>"` — Cobra interprets angle brackets as a literal command name, not a placeholder, so `cortex ticket show <uuid>` never matched

## Resolution

Single commit (`6d65d29`) across 5 files:

- `cmd/cortex/commands/ticket_show.go` — Fixed `Use: "<id>"` → `Use: "show <id>"` so Cobra routes correctly
- `cmd/cortex/commands/ticket.go` — Added Run handler to show help when no subcommand given
- `internal/daemon/api/tickets.go` — Popup command: `cortex ticket show %s`
- `internal/core/spawn/spawn.go` (2 locations) — Companion cmd: `cortex ticket show %s` (removed redundant `CORTEX_TICKET_ID` env var prefix)
- `internal/cli/sdk/client.go` — Updated comment

All verification passed: build, lint, unit tests. Merged to main via fast-forward.