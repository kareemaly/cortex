---
id: 0a9c4d7d-21b0-494b-a791-56a7538c970a
title: MCP Spawn Should Delegate to Daemon HTTP API
type: ""
created: 2026-01-27T09:40:53.274096Z
updated: 2026-01-27T10:14:39.943679Z
---
## Problem

The MCP spawn handler in `tools_architect.go` calls `spawn.Orchestrate()` directly rather than delegating to the daemon HTTP API (`POST /tickets/{status}/{id}/spawn`).

This means the MCP stdio process (spawned by Claude Code at session start) uses its own compiled-in version of the orchestration logic. When the daemon binary is updated and restarted, the MCP process still runs old code — causing issues like config parsing failures when the YAML schema changes.

### Current Flow
```
MCP stdio process → spawn.Orchestrate() (compiled into MCP binary)
HTTP daemon        → spawn.Orchestrate() (compiled into daemon binary)
```

### Expected Flow
```
MCP stdio process → HTTP API (POST /tickets/{status}/{id}/spawn) → daemon handles it
```

## Solution

Refactor the MCP spawn handler to call the daemon HTTP API instead of invoking `spawn.Orchestrate()` directly. The MCP process already knows the daemon port — it should act as a thin client that forwards spawn requests to the HTTP daemon.

This way:
- Only one process (the daemon) owns the spawn logic
- Updating and restarting the daemon immediately affects all MCP sessions
- No need to restart Claude Code sessions after daemon upgrades

## Scope

- **`internal/daemon/mcp/tools_architect.go`** — Replace direct `spawn.Orchestrate()` call with HTTP client call to `POST /tickets/{status}/{id}/spawn`
- The MCP handler should only pass `ticket_id` and optional `mode` — the daemon resolves the agent from project config via the existing fallback chain (`request → project config → "claude"`)
- Remove the `agent` field from `SpawnSessionInput` in `types.go` since it's unnecessary when delegating to the daemon
- Ensure error responses from the HTTP API are properly surfaced back through MCP

## Acceptance Criteria

- [ ] MCP spawn handler delegates to daemon HTTP API instead of calling `spawn.Orchestrate()` directly
- [ ] MCP spawn tool only accepts `ticket_id` and optional `mode` (no `agent` param)
- [ ] Spawn behavior is identical from both MCP and HTTP paths
- [ ] Restarting the daemon picks up new spawn logic for existing MCP sessions
- [ ] Error handling preserved — orphaned session errors, config errors, etc. still surface correctly