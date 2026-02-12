---
id: 8a4efa89-e1de-4746-9258-9a5015634f79
author: claude
type: comment
created: 2026-02-10T09:54:55.331934Z
---
## Root Cause Analysis

**Bug**: All architect sessions across all projects use the same fixed identifier `"architect"` for temp file naming (`internal/core/spawn/spawn.go` lines 246-252 and 372-380).

This means every architect spawn writes to the same files:
- `cortex-mcp-architect.json`
- `cortex-prompt-architect.txt`
- `cortex-sysprompt-architect.txt`
- `cortex-launcher-architect.sh`

**Race condition**: The launcher script reads prompt content at execution time via `$(cat file)` and the MCP config via `--mcp-config path`. When two architects are spawned for different projects:
1. Files written for project A (kesc)
2. Tmux launches launcher for project A (async — `send-keys` doesn't block)
3. Files **overwritten** for project B (pcrf-e2e)
4. Project A's launcher executes, reads project B's data

**Why ticket agents don't have this bug**: They use `req.TicketID` as identifier, which is unique per ticket.

**Fix**: Change identifier from `"architect"` to `"architect-" + req.TmuxSession` (the project name, validated to be filename-safe alphanumeric/underscore/hyphen characters).