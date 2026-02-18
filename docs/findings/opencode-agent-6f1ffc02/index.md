---
id: 6f1ffc02-8b0b-43ee-ace7-2e7c1bfc2e3c
title: OpenCode Agent Status Hooks — Research Findings
tags:
    - opencode
    - hooks
    - agent-status
    - integration
    - research
created: 2026-02-13T10:31:42.48479Z
updated: 2026-02-13T10:31:42.48479Z
---
## Executive Summary

Claude Code provides a first-class hook system that Cortex leverages for real-time agent status tracking (idle, in_progress, waiting_permission). OpenCode has **equivalent and even richer** mechanisms — a plugin hook system, an HTTP server with SSE events, and a TypeScript SDK — but Cortex currently uses **none of them**. OpenCode sessions are stuck at "starting" status forever because no status feedback path is wired up.

---

## 1. How Claude Code Status Hooks Work in Cortex Today

### Architecture

```
Claude Code Agent
    │
    ├─ PostToolUse hook ──→ cortexd hook post-tool-use ──→ POST /agent/status {status: "in_progress", tool: "..."}
    ├─ Stop hook ──────────→ cortexd hook stop ───────────→ POST /agent/status {status: "idle"}
    └─ PermissionRequest ──→ cortexd hook permission-req → POST /agent/status {status: "waiting_permission"}
                                                                    │
                                                              SessionStore.UpdateStatus()
                                                                    │
                                                              .cortex/sessions.json (persisted)
```

### Key Files

| File | Role |
|------|------|
| `internal/core/spawn/settings.go` | Generates `cortex-settings-{id}.json` with hook definitions |
| `cmd/cortexd/commands/hook.go` | CLI commands that receive hook callbacks and POST to daemon |
| `internal/daemon/api/agent.go` | `POST /agent/status` endpoint — validates and stores status |
| `internal/session/session.go` | Defines 5 `AgentStatus` values: starting, in_progress, idle, waiting_permission, error |
| `internal/session/store.go` | `UpdateStatus()` persists status + tool to sessions.json |

### How It Works

1. **At spawn time** (`spawn.go:260-274`): For non-OpenCode agents, a `cortex-settings-{id}.json` file is generated with three hook entries:
   - `PostToolUse` (matcher: `*`) → `cortexd hook post-tool-use`
   - `Stop` → `cortexd hook stop`
   - `PermissionRequest` → `cortexd hook permission-request`

2. **Claude Code invokes hooks**: When Claude Code uses a tool, stops, or requests permission, it executes the configured shell commands. These commands receive JSON on stdin (e.g., `{"tool_name": "Write"}`).

3. **Hook handler** (`hook.go`): Reads `CORTEX_TICKET_ID` and `CORTEX_PROJECT` from env vars (set at spawn), parses stdin JSON, and POSTs to `http://localhost:4200/agent/status`.

4. **Daemon stores status** (`agent.go`): Validates the session exists, calls `SessionStore.UpdateStatus()`, which persists to `.cortex/sessions.json`.

### Key Design Decisions

- **Push-based**: Hooks push status → no polling needed
- **Fail-gracefully**: All hook handlers return nil on error (never block the agent)
- **5-second timeout**: Hook HTTP calls timeout after 5s to avoid stalling Claude Code
- **Env-var routing**: `CORTEX_TICKET_ID` + `CORTEX_PROJECT` identify which session to update

### Gap: No SSE Emission

The event bus defines `SessionStatus` as an event type (`internal/events/bus.go:15`), but `agent.go` never emits it after updating status. TUI clients must poll `sessions.json` rather than receiving real-time SSE updates.

---

## 2. Current OpenCode Integration in Cortex

### What Exists Today

- **Config injection** (`opencode_config.go`): `OPENCODE_CONFIG_CONTENT` env var passes agent definition + MCP servers
- **Permission bypass**: `"permission": {"*": "allow"}` auto-approves all tools
- **MCP tools**: Cortex MCP tools (readReference, addComment, etc.) work via the MCP server config
- **No hooks**: Settings/hooks generation is explicitly skipped for OpenCode (`spawn.go:262`: `if req.Agent != "opencode"`)

### What's Missing

- Status never transitions from `starting` — no feedback mechanism is configured
- No detection of idle, in_progress, or waiting_permission states
- No way for the TUI companion pane to show meaningful agent status

---

## 3. What OpenCode Offers for Status Integration

OpenCode has **three distinct mechanisms** that could provide agent status awareness:

### 3a. Plugin Hook System (Most Analogous to Claude Code Hooks)

OpenCode supports JavaScript/TypeScript plugins in `.opencode/plugins/`. These run in-process and can hook into lifecycle events.

**Relevant hooks:**

| Hook | Maps to Cortex Status |
|------|----------------------|
| `session.idle` | `idle` |
| `session.error` | `error` |
| `tool.execute.before` | `in_progress` (+ tool name) |
| `tool.execute.after` | `in_progress` (completed tool) |
| `permission.asked` | `waiting_permission` |
| `permission.replied` | `in_progress` (permission granted) |
| `stop` | `idle` (agent terminating) |

**Example plugin** (`.opencode/plugins/cortex-status.ts`):
```typescript
import type { Plugin } from "opencode/plugin"

const plugin: Plugin = {
  name: "cortex-status",
  setup(app) {
    const baseUrl = process.env.CORTEX_DAEMON_URL || "http://localhost:4200"
    const ticketId = process.env.CORTEX_TICKET_ID
    const project = process.env.CORTEX_PROJECT

    async function postStatus(status: string, tool?: string) {
      if (!ticketId || !project) return
      try {
        await fetch(`${baseUrl}/agent/status`, {
          method: "POST",
          headers: { "Content-Type": "application/json", "X-Cortex-Project": project },
          body: JSON.stringify({ ticket_id: ticketId, status, tool }),
          signal: AbortSignal.timeout(5000),
        })
      } catch {} // fail gracefully
    }

    app.on("tool.execute.before", async (event) => {
      await postStatus("in_progress", event.tool?.name)
    })

    app.on("session.idle", async () => {
      await postStatus("idle")
    })

    app.on("permission.asked", async () => {
      await postStatus("waiting_permission")
    })

    app.on("session.error", async () => {
      await postStatus("error")
    })
  }
}

export default plugin
```

**Pros:**
- Closest equivalent to Claude Code's hook model
- Runs in-process, low latency
- Access to rich event data (tool name, permission details)
- Can be injected via config or file

**Cons:**
- Requires plugin file to exist in the project's `.opencode/plugins/` directory
- Plugin API may evolve (still relatively new)
- Need to verify `OPENCODE_CONFIG_CONTENT` can define plugins or if they must be file-based

### 3b. HTTP Server + SSE Events (`opencode serve`)

OpenCode can run as a headless HTTP server that streams all events over SSE.

**Key endpoints:**

| Endpoint | Purpose |
|----------|---------|
| `GET /global/event` | SSE stream of all events |
| `GET /session/:id` | Session state |
| `POST /session/:id/message` | Send message (synchronous — blocks until done) |

**SSE event types:** `session.idle`, `session.error`, `session.created`, `session.updated`, `message.updated`, `message.part.updated`

**Architecture:**
```
Cortex Daemon
    │
    ├─ Spawn: `opencode serve --port {dynamic}`
    │
    ├─ Subscribe: GET /global/event (SSE)
    │   ├─ session.idle → POST /agent/status {status: "idle"}
    │   ├─ session.error → POST /agent/status {status: "error"}
    │   └─ message.updated → POST /agent/status {status: "in_progress"}
    │
    └─ Send work: POST /session/:id/message
```

**Pros:**
- Rich, real-time event stream
- Language-agnostic (HTTP/SSE)
- Could manage OpenCode programmatically (send messages, read state)
- Authentication via `OPENCODE_SERVER_PASSWORD`

**Cons:**
- Requires running OpenCode in server mode (headless) rather than TUI mode
- Would need a fundamentally different spawn model (no tmux TUI)
- More complex architecture (daemon manages OpenCode HTTP server lifecycle)
- No `permission.asked` event in SSE (permissions are auto-approved in non-interactive mode)

### 3c. Tmux Output Parsing (Fallback)

If OpenCode is running in TUI mode (current approach), status could be inferred from terminal output.

**Detectable patterns:**
- Tool execution indicators in the TUI
- Permission prompts (if not bypassed)
- Idle state (cursor blinking, no activity)

**Pros:**
- Works with current spawn model (no changes)
- Agent-agnostic (could work for any TUI agent)

**Cons:**
- Fragile — depends on TUI output format which can change
- Requires periodic polling via `tmux capture-pane`
- Limited information (no tool names, no structured data)
- High maintenance burden

---

## 4. Recommended Approach

### Primary: Plugin-Based Status Reporting (3a)

This is the recommended approach because:

1. **Minimal architecture change** — keeps current TUI-in-tmux spawn model
2. **Symmetric with Claude Code** — same push-based status updates to the same `/agent/status` endpoint
3. **Rich data** — tool names, permission events, error details
4. **Simple implementation** — a single TypeScript plugin file

### Implementation Steps

1. **Create plugin file**: Write a `cortex-status.ts` plugin that posts to `/agent/status`
2. **Inject plugin at spawn time**: Either:
   - Write the plugin to a temp file and reference it in `OPENCODE_CONFIG_CONTENT`
   - Write it to `.opencode/plugins/` in the working directory
   - Investigate if `OPENCODE_CONFIG_CONTENT` supports inline plugin definitions
3. **Remove permission bypass** (optional): With `permission.asked` → `waiting_permission` support, Cortex could surface permission prompts to the architect rather than auto-approving everything
4. **Wire SSE emission**: Fix the existing gap — emit `SessionStatus` events on the bus when status changes, so the TUI gets real-time updates for both agents

### Secondary: Server Mode for Advanced Orchestration

For future consideration, `opencode serve` could enable:
- Programmatic message sending (Cortex daemon sends follow-up instructions)
- Session state inspection without tmux
- Multi-agent coordination

This would require a larger architectural change and could be a separate initiative.

### Not Recommended: Tmux Parsing

Too fragile and maintenance-heavy. Only consider as an absolute last resort if plugins are not viable.

---

## 5. Open Questions

1. **Plugin injection via env var**: Can `OPENCODE_CONFIG_CONTENT` define plugins, or must they be files in `.opencode/plugins/`? If file-only, the spawn process needs to write the plugin file to the working directory.

2. **Permission mode trade-off**: Currently `"*": "allow"` bypasses all permissions. With status hooks, should Cortex instead use `"*": "ask"` and surface permission prompts? This would give architects visibility into what tools agents are requesting.

3. **SSE event gap**: The `SessionStatus` event type exists but is never emitted. Should this be fixed as part of this work, or tracked separately?

4. **OpenCode version pinning**: The plugin API is relatively new. Should Cortex pin to a minimum OpenCode version that supports the required hooks?

5. **Error recovery**: If the plugin fails to post status (daemon unreachable), should it retry? Claude Code's hooks use a simple fire-and-forget model with a 5-second timeout.

---

## 6. Summary of Agent Status Capabilities

| Capability | Claude Code | OpenCode |
|-----------|-------------|----------|
| Push-based status hooks | Yes (settings.json hooks) | Yes (plugin hooks) |
| Tool execution events | PostToolUse (tool_name on stdin) | tool.execute.before/after |
| Stop/idle detection | Stop hook | session.idle hook |
| Permission waiting | PermissionRequest hook | permission.asked hook |
| Error detection | Not exposed | session.error hook |
| HTTP API for control | No | Yes (opencode serve) |
| SSE event stream | No | Yes (GET /global/event) |
| SDK for integration | No | Yes (@opencode-ai/sdk) |
| Current Cortex integration | Full (3 status states) | None (stuck at "starting") |