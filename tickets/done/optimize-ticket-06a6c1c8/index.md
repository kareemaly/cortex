---
id: 06a6c1c8-19da-4fea-b03a-2872676e20f7
title: Optimize ticket agent tools and prompts
type: work
tags:
    - mcp
    - session
    - research
created: 2026-02-10T07:56:01.69532Z
updated: 2026-02-10T08:19:43.571426Z
---
## Problem

Ticket agents have tool and prompt issues causing wasted tokens, plan confusion, and limited cross-referencing:

1. **Redundant `readTicket`**: The KICKOFF prompt already injects the full ticket title/body, but all SYSTEM prompts instruct the agent to "read the ticket" as step 1. Agents immediately call `readTicket`, duplicating tokens. When agents resume after plan approval, re-reading the original ticket causes confusion if the approved plan deviates from the original scope.

2. **No cross-referencing**: Tickets can reference other tickets or docs (e.g., `ticket:abc123`, `doc:xyz789`), but agents have no tools to follow those references.

3. **Research output channel**: Research agents are instructed to document findings via `addComment`, which is noisy and ephemeral. Research output should be docs, not comments.

## Changes Required

### Tool changes (`internal/daemon/mcp/tools_ticket.go`)

- **Remove `readTicket`** — no longer needed since ticket content is injected in KICKOFF
- **Add `readReference`** — new tool to read a referenced ticket or doc
  - Params: `id` (required, string), `type` (required, string: "ticket" or "doc")
  - Returns the full ticket or doc content
  - Available to all ticket types
- **Add `createDoc`** — available to **research ticket type only**
  - Params: `title` (required), `category` (required), `body` (optional), `tags` (optional)
  - Research agents should produce docs as their primary deliverable

### Prompt changes (all under `internal/install/defaults/claude-code/prompts/ticket/`)

- **All SYSTEM prompts**: Remove "read the ticket" as a first step. Update MCP tool list to reflect new tools (remove `readTicket`, add `readReference`). Add a note about using `readReference` to follow cross-references mentioned in the ticket.
- **Research SYSTEM prompt**: Add `createDoc` to tool list. Change guidance from "document findings via `addComment`" to "create docs for findings, use `addComment` only for brief progress updates."

## Acceptance criteria

- `readTicket` tool is removed from ticket agent tool set
- `readReference` tool works for both ticket and doc types
- `createDoc` is available only when ticket type is "research"
- All four SYSTEM prompts updated (work, debug, research, chore)
- Research SYSTEM prompt guides agent to use `createDoc` for findings
- Existing tests pass; new tool tests added