---
id: 9d3af9b2-0638-4798-b1ef-e760c5509d1a
author: claude
type: review_requested
created: 2026-02-10T08:18:05.578563Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/0ac20478-e66c-4adf-ac0f-636de072dc64
        commit: 9f20a34
---
## Summary

Optimized ticket agent tools and prompts to reduce token waste and enable cross-referencing and research doc creation.

### Changes

**Tool changes (`tools_ticket.go`)**:
- Removed `readTicket` tool and `EmptyInput`/`handleReadOwnTicket` (ticket content already injected via KICKOFF prompt)
- Added `readReference` tool — reads referenced tickets or docs by ID and type ("ticket"/"doc"), available to all ticket types
- Added `createDoc` tool — creates documentation files, conditionally registered only for research ticket types

**Ticket type threading**:
- Added `TicketType` to `MCPConfigParams` (spawn/config.go) → passes `--ticket-type` CLI flag
- Added `TicketType` to `ResumeRequest` (spawn/spawn.go) and `orchestrate.go` resume path
- Added `--ticket-type` flag to `cortexd mcp` command (cmd/cortexd/commands/mcp.go)
- Added `TicketType` to MCP `Config` and `Session` structs (server.go, types.go)

**New types** (types.go):
- `ReadReferenceInput` / `ReadReferenceOutput` — union output with optional Ticket/Doc fields
- `TicketCreateDocInput` — simplified createDoc input without cross-project support

**Prompt updates** (8 files):
- All 4 claude-code SYSTEM.md prompts: replaced `readTicket` with `readReference`, updated workflow steps
- All 4 copilot KICKOFF.md prompts: updated tool tables, replaced `readTicket` row with `readReference`
- Research prompts: added `createDoc` tool, changed guidance to "create docs" instead of "document via addComment"

**Config updates** (cortex.yaml):
- Updated `--allowedTools` for all 4 ticket types: `readTicket` → `readReference`
- Research type now includes `mcp__cortex__createDoc`

**Tests**:
- Added `TestNewServerTicketWithType` and `TestNewServerTicketDefaultType` (server_test.go)
- Replaced `TestHandleReadOwnTicket` with `TestHandleReadReference_Ticket`, `TestHandleReadReference_InvalidType`, `TestHandleReadReference_EmptyID` (tools_test.go)
- Updated `TestGenerateMCPConfig_WithTicket` to include TicketType, added `TestGenerateMCPConfig_WithTicketNoType` (spawn_test.go)

All 19 files changed, `make build` / `make test` / `make lint` all pass clean.