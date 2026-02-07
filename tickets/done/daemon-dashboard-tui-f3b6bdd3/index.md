---
id: f3b6bdd3-9a7b-44a3-9393-b4e5ebba3b9a
title: Daemon Dashboard TUI
type: ""
created: 2026-01-27T10:40:00.537456Z
updated: 2026-01-27T12:15:38.97823Z
---
## Problem

There is no central command center for managing all cortex projects. Users must navigate individual project sessions manually. The daemon (`cortexd`) runs headless with raw log output.

## Solution

Two new CLI commands that together form the cortex command center:

### 1. `cortex start` — Orchestrator

Creates (or attaches to) a `CortexDaemon` tmux session with two panes:

```
CortexDaemon tmux session
┌──────────────────────────────┬───────────────┐
│                              │               │
│  cortex dashboard            │   cortexd     │
│  (client TUI, 70%)          │   (logs, 30%) │
│                              │               │
└──────────────────────────────┴───────────────┘
```

Logic:
1. Check if `CortexDaemon` tmux session exists
2. If yes — attach/switch to it
3. If no — create session, run `cortexd` in initial pane, split left (70%) for `cortex dashboard`

### 2. `cortex dashboard` — Client TUI

Standalone bubbletea TUI that calls daemon HTTP APIs. Displays a navigable tree of all registered projects with nested active sessions.

```
┌──────────────────────────────────────────────────┐
│  Cortex Dashboard                                │
├──────────────────────────────────────────────────┤
│                                                  │
│  ▼ Cortex (3 backlog · 1 prog · 0 review)       │
│    ● architect              running     2h 15m   │
│    ● fix-auth-bug           progress       45m   │
│  ▼ My App (12 backlog · 2 prog · 1 review)      │
│    ● architect              running     1h 30m   │
│    ○ add-dark-mode          review         12m   │
│  ▶ Backend API (5 backlog · 0 prog · 0 review)  │
│                                                  │
├──────────────────────────────────────────────────┤
│  [f]ocus  [s]pawn architect  [q]uit              │
└──────────────────────────────────────────────────┘
```

**Tree nodes:**
- **Project header** — expandable/collapsible, shows aggregated ticket counts by status
- **Session row** — leaf node, shows ticket/window name + status + duration since session started

**Keybindings:**
- `j/k` — navigate tree up/down
- `enter` / `l` — expand/collapse project
- `f` — focus session: `tmux select-window` to jump to that agent's tmux window
- `s` — spawn architect session for the selected project
- `q` — quit dashboard

**Data flow:**
- Reads project list from global registry (`~/.cortex/settings.yaml` `projects` list)
- For each project: fetches ticket counts via `GET /tickets` with `X-Cortex-Project` header
- Active sessions: tickets in progress/review where `has_active_session: true`
- SSE subscription per project for real-time updates (ticket moves refresh counts and session list)

## Architecture

**Key design principle: dashboard is a pure client.** It uses the same HTTP API as CLI, MCP, and TUI. The daemon stays headless — no TUI complexity in the daemon process.

Benefits:
- Dashboard can crash/restart without killing the daemon
- Clean separation of concerns
- Multiple dashboards could run simultaneously
- Dashboard is testable against the HTTP API

## Dependencies

- **Global Project Registry** (`~/.cortex/settings.yaml` `projects` list) — needed to know which projects exist
- **`GET /projects` endpoint** — daemon API to list registered projects with basic stats

## Scope

### `cortex start` command (`cmd/cortex/commands/start.go`)
- Create/attach to `CortexDaemon` tmux session
- Split panes: left 70% dashboard, right 30% daemon
- Minimal command — just tmux orchestration

### `cortex dashboard` command (`cmd/cortex/commands/dashboard.go`)
- New bubbletea TUI in `internal/cli/tui/dashboard/`
- Tree model with project headers and session rows
- HTTP client calls to daemon for data
- SSE subscription for real-time updates
- Keybindings: j/k navigation, enter expand/collapse, f focus, s spawn, q quit

### Daemon changes
- None for the daemon itself (stays headless)
- Depends on `GET /projects` endpoint from the registry ticket

## Acceptance Criteria

- [ ] `cortex start` creates CortexDaemon tmux session with dashboard + daemon panes
- [ ] `cortex start` attaches to existing session if already running
- [ ] `cortex dashboard` shows all registered projects in a tree view
- [ ] Project headers display ticket counts (backlog, progress, review)
- [ ] Active sessions listed under each project with status and duration
- [ ] `f` key focuses (jumps to) the selected session's tmux window
- [ ] `s` key spawns an architect session for the selected project
- [ ] Tree updates in real-time via SSE when tickets move
- [ ] `j/k` navigation and `enter` expand/collapse work correctly