---
id: e1d73a71-cdf1-4d22-937f-7b9b327fe072
title: Track architect session with status, resume/fresh, and concludeSession
type: work
tags:
    - session
    - architect
    - tui
    - mcp
    - spawn
created: 2026-02-07T11:23:48.158475Z
updated: 2026-02-07T12:04:12.796466Z
---
## Problem

Architect sessions are untracked — no record in `.cortex/sessions.json`, no status/tool tracking, no orphan detection, no resume support. The only state detection is checking if a tmux window named "architect" exists.

## Requirements

### 1. Persist architect session in session store

- Store architect session in `.cortex/sessions.json` under a special key (e.g. `"architect"`)
- Track: agent name, tmux window, started_at, status, tool (same fields as ticket agent sessions)
- Remove the `if req.AgentType == AgentTypeTicketAgent` guard in `internal/core/spawn/spawn.go` (or extend it to also handle architect)

### 2. Spawn logic with state detection

When spawning architect (`POST /architect/spawn` and `cortex architect`):

| State | Behavior |
|-------|----------|
| **No session, no window** | Spawn new (normal) |
| **Session exists + window exists** | Focus the existing window |
| **Session exists + window gone (orphaned)** | Support `fresh` / `resume` mode selection |

- `cortex architect` CLI should accept a mode flag or prompt for fresh/resume when orphaned
- Dashboard TUI should present fresh/resume choice when orphaned

### 3. `concludeSession` MCP tool for architect

- New MCP tool available to **architect agent** (in `tools_architect.go`)
- Deletes the architect session from session store
- Destroys the architect tmux window
- This is how an architect cleanly ends its session

### 4. Dashboard TUI — architect status display

- Show architect session status in the **dashboard TUI** (not kanban)
- Display: agent status (starting/in_progress/idle/waiting_permission/error), current tool, session duration
- Show orphaned state if applicable

### 5. Include architect in listSessions

- `GET /sessions` and `listSessions` MCP tool should include the architect session alongside ticket agent sessions
- Distinguish it from ticket sessions (no ticket ID, marked as architect type)

## Key Files

- Session store: `internal/session/store.go`, `internal/session/session.go`
- Spawn logic: `internal/core/spawn/spawn.go`, `internal/core/spawn/orchestrate.go`, `internal/core/spawn/state.go`
- Architect handlers: `internal/daemon/api/architect.go`
- Session handlers: `internal/daemon/api/sessions.go`
- MCP architect tools: `internal/daemon/mcp/tools_architect.go`
- Architect CLI: `cmd/cortex/commands/architect.go`
- Dashboard TUI: `internal/cli/tui/`
- Response types: `internal/types/response.go`
- SessionManager: `internal/daemon/api/session_manager.go`