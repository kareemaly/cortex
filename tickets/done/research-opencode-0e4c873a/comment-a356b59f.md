---
id: a356b59f-59d1-4810-b41d-906d1676e715
author: claude
type: review_requested
created: 2026-02-13T10:31:55.480652Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Research Complete: OpenCode Agent Status Hooks

### Key Findings

**Claude Code integration (working today):**
- Uses `settings.json` hooks (PostToolUse, Stop, PermissionRequest) → `cortexd hook` CLI → `POST /agent/status`
- Push-based, fail-graceful, 5-second timeout. Three status transitions: in_progress, idle, waiting_permission.
- One gap: `SessionStatus` SSE event is defined but never emitted — TUI must poll.

**OpenCode integration (broken today):**
- Status is stuck at "starting" — no feedback mechanism is wired up (`spawn.go:262` explicitly skips hooks for OpenCode)
- Config injection and MCP tools work, but agent status is a black hole.

**OpenCode capabilities (available but unused):**
1. **Plugin hook system** — JS/TS plugins with `session.idle`, `tool.execute.before`, `permission.asked`, `session.error` hooks
2. **HTTP server + SSE** — `opencode serve` with real-time event streaming
3. **TypeScript SDK** — `@opencode-ai/sdk` wrapping the HTTP API

### Recommendation

**Use the plugin hook system** — it's the closest equivalent to Claude Code's hooks. A single `cortex-status.ts` plugin can POST to `/agent/status` on the same events, using the same env vars (`CORTEX_TICKET_ID`, `CORTEX_PROJECT`, `CORTEX_DAEMON_URL`) already injected at spawn time. Minimal architecture change, same push-based model.

### Open Questions for Decision
1. Can `OPENCODE_CONFIG_CONTENT` define plugins, or must they be file-based?
2. Should we reconsider the `"*": "allow"` permission bypass now that we can detect `permission.asked`?
3. Should the `SessionStatus` SSE gap be fixed as part of this work?

Full findings documented in: "OpenCode Agent Status Hooks — Research Findings"