---
id: 072bb2e6-1eb0-4de0-895b-d992e877e8f9
author: claude
type: done
created: 2026-02-10T08:19:43.570255Z
---
## Completed: Optimize ticket agent tools and prompts

### What was done
Removed redundant `readTicket` tool from ticket agents, added `readReference` for cross-referencing, and added `createDoc` for research agents.

### Changes (20 files, 287 additions, 78 deletions)

**Core tool changes:**
- Removed `readTicket` tool, `EmptyInput` struct, and `handleReadOwnTicket` handler from ticket agents
- Added `readReference` tool — reads referenced tickets or docs by ID/type, available to all ticket types
- Added `createDoc` tool — conditionally registered only for research ticket types

**Ticket type threading (new plumbing):**
- `MCPConfigParams.TicketType` → `--ticket-type` CLI flag → `mcp.Config.TicketType` → `Session.TicketType`
- Threaded through `spawn.go` (Spawn + Resume), `orchestrate.go`, and `mcp.go` CLI command

**New types:** `ReadReferenceInput`, `ReadReferenceOutput`, `TicketCreateDocInput`

**Prompt updates (8 files):**
- All claude-code SYSTEM.md and copilot KICKOFF.md prompts updated with new tool references and workflows
- Research prompts now guide agents to use `createDoc` for findings instead of `addComment`

**Config:** Updated `--allowedTools` in cortex.yaml for all 4 ticket types

**Tests:** Added 5 new tests, updated 1 existing test

**Docs:** Updated CLAUDE.md ticket agent tools table

All checks pass: `make build`, `make test`, `make lint`, pre-push hooks.