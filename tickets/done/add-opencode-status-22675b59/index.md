---
id: 22675b59-01e4-4010-907f-027e4a24a983
title: Add OpenCode status plugin injection at spawn time
type: work
tags:
    - opencode
    - agent
    - hooks
references:
    - doc:5208ef3a-80f4-4da4-a1ce-217c05bc1eb5
    - doc:6f1ffc02-8b0b-43ee-ace7-2e7c1bfc2e3c
    - doc:e1c47258-a082-4b06-92ca-683cc5cad8b0
created: 2026-02-13T13:24:56.100429Z
updated: 2026-02-13T13:37:33.83181Z
---
## Problem

OpenCode agent sessions have no status feedback — status is permanently stuck at `starting` because hook generation is explicitly skipped for OpenCode. The TUI cannot show whether an OpenCode agent is working, idle, waiting for permission, or errored.

## Solution

Inject a Cortex status plugin into OpenCode at spawn time using a session-unique temp directory and the `OPENCODE_CONFIG_DIR` environment variable. The plugin pushes status updates to the existing `POST /agent/status` endpoint using the same env vars already injected at spawn (`CORTEX_TICKET_ID`, `CORTEX_PROJECT`, `CORTEX_DAEMON_URL`).

## Requirements

### Plugin Injection

- At spawn time, create a session-unique temp directory (e.g., `/tmp/cortex-opencode-{sessionID}/`)
- Write `plugin/cortex-status.ts` inside it — a minimal TypeScript plugin with no imports (untyped exports to avoid triggering OpenCode's 3-second `@opencode-ai/plugin` auto-install)
- Set `OPENCODE_CONFIG_DIR` env var to the temp directory in the launcher
- Make the plugin directory read-only after writing to skip dependency install overhead
- Register the temp directory in `CleanupFiles` for automatic cleanup on session end

### Plugin Event Mapping

The plugin should handle these OpenCode events and POST to `POST /agent/status`:

| OpenCode Event | → Cortex Status |
|---|---|
| `session.status` → `busy` | `in_progress` |
| `session.status` → `idle` | `idle` |
| `session.status` → `retry` | `error` |
| `session.idle` | `idle` |
| `permission.asked` | `waiting_permission` |
| `permission.replied` | `in_progress` |
| `tool.execute.before` | `in_progress` (with tool name if available) |
| `tool.execute.after` | `in_progress` |

### Plugin Design

- Fire-and-forget HTTP POSTs with 5-second timeout (same pattern as Claude Code hooks)
- Fail gracefully — never block the agent if daemon is unreachable
- Daemon URL, ticket ID, and project path baked into the plugin at generation time via string replacement
- Plugin posts to the existing `POST /agent/status` endpoint with `X-Cortex-Project` header

### Spawn Changes

- Modify OpenCode spawn path in `internal/core/spawn/spawn.go` to create the temp dir, write the plugin, and set `OPENCODE_CONFIG_DIR`
- Keep `"*": "allow"` permission model unchanged

## References

- Findings doc: "OpenCode Plugin Injection via Config & Temp Directory" (doc:5208ef3a)
- Findings doc: "OpenCode Agent Status Hooks — Research Findings" (doc:6f1ffc02)
- Findings doc: "Cortex Agent Status Integration Architecture" (doc:e1c47258)

## Acceptance Criteria

- OpenCode sessions report real-time status updates (no longer stuck at `starting`)
- Plugin is written to a unique temp dir per session and cleaned up on session end
- Status transitions work: idle, in_progress (with tool name), waiting_permission, error
- Existing `POST /agent/status` endpoint receives updates from the plugin
- No changes to OpenCode's permission model
- Build, lint, and tests pass