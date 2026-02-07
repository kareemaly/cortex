---
id: b138d578-9dbf-4dd5-a622-21c257875fe4
title: CLI Restructure
type: ""
created: 2026-01-23T07:12:57Z
updated: 2026-01-23T07:12:57Z
---
Restructure CLI commands under `cortex ticket` subcommand and add mode flags.

## Context

This is a fresh project with no users. No backward compatibility needed. Breaking changes are fine. Do not accumulate tech debt.

## Command Changes

### Before
```
cortex list [--status] [--json]
cortex spawn <ticket-id> [--json]
cortex session <ticket-id> [--json]
cortex architect [--detach]
```

### After
```
cortex ticket list [--status] [--query] [--json]
cortex ticket spawn <id> [--resume] [--fresh] [--json]
cortex ticket <id> [--json]                          # shows ticket details (replaces session)
cortex architect [--resume] [--fresh] [--detach]
```

## Details

### `cortex ticket` subcommand
Create parent command that groups ticket operations.

### `cortex ticket list`
- Add `--query` flag for text search (matches API)
- Keep `--status` and `--json` flags

### `cortex ticket spawn <id>`
- Add `--resume` flag (mode=resume for orphaned sessions)
- Add `--fresh` flag (mode=fresh for orphaned sessions)
- Default mode=normal

### `cortex ticket <id>`
- Replaces `cortex session` command
- Shows ticket details with session info
- Keep `--json` flag

### `cortex architect`
- Add `--resume` flag
- Add `--fresh` flag
- Keep `--detach` flag
- Uses SDK `SpawnArchitect(mode)`

## Files to Change

- `cmd/cortex/commands/list.go` → `cmd/cortex/commands/ticket_list.go`
- `cmd/cortex/commands/spawn.go` → `cmd/cortex/commands/ticket_spawn.go`
- `cmd/cortex/commands/session.go` → `cmd/cortex/commands/ticket_show.go`
- `cmd/cortex/commands/ticket.go` (new) - parent command
- `cmd/cortex/commands/architect.go` - add flags
- `cmd/cortex/commands/root.go` - update command registration

## Verification

```bash
make lint
make test
make build
make test-integration

# Manual verification
cortex ticket list --status backlog
cortex ticket list --query "search term"
cortex ticket spawn <id> --resume
cortex architect --resume
```

## Implementation

### Commits Pushed
- `f4977fd` feat: restructure CLI under ticket subcommand with mode flags

### Key Files Changed
- `cmd/cortex/commands/ticket.go` - New parent command for ticket operations
- `cmd/cortex/commands/ticket_list.go` - List command with --query flag (renamed from list.go)
- `cmd/cortex/commands/ticket_spawn.go` - Spawn command with --resume/--fresh flags (renamed from spawn.go)
- `cmd/cortex/commands/ticket_show.go` - Show ticket details (renamed from session.go)
- `cmd/cortex/commands/architect.go` - Simplified to use SDK, added --resume/--fresh flags
- `internal/cli/sdk/client.go` - Added query parameter to ListAllTickets/ListTicketsByStatus
- `internal/daemon/api/tickets.go` - Added query parameter support to list handlers
- `internal/daemon/api/types.go` - Added filterSummaryList helper with query filtering

### Important Decisions
- Architect command now uses SDK `SpawnArchitect(mode)` instead of direct tmux logic, simplifying the CLI code significantly
- Query filter is case-insensitive and matches against both title and body (same as MCP handler)
- Removed unused `toSummaryList` function, replaced with `filterSummaryList` that handles query filtering

### Scope Changes
- No changes from original ticket scope