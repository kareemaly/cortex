---
id: 25be49a5-3746-4fe8-ae80-8a46e83ee301
author: claude
type: done
created: 2026-02-13T12:49:56.843682Z
---
## Research Complete: OpenCode Agent Status Hooks

### Key Findings

1. **Claude Code integration (working)**: Uses `settings.json` hooks (PostToolUse → in_progress, Stop → idle, PermissionRequest → waiting_permission) that execute `cortexd hook` CLI commands, which POST to `POST /agent/status`. Push-based, fail-graceful with 5s timeout. Key files: `internal/core/spawn/settings.go`, `cmd/cortexd/commands/hook.go`, `internal/daemon/api/agent.go`.

2. **OpenCode integration (broken)**: Status permanently stuck at "starting". Hook generation explicitly skipped at `spawn.go:262` (`if req.Agent != "opencode"`). Config injection and MCP tools work, but no status feedback path exists.

3. **OpenCode has rich, unused capabilities**:
   - **Plugin hook system**: JS/TS plugins in `.opencode/plugins/` with hooks for `session.idle`, `tool.execute.before/after`, `permission.asked/replied`, `session.error`, `stop`
   - **HTTP server + SSE**: `opencode serve` with `/global/event` SSE stream and synchronous `/session/:id/message` endpoint
   - **TypeScript SDK**: `@opencode-ai/sdk` wrapping the HTTP API

4. **Existing gap**: `SessionStatus` event type defined in `internal/events/bus.go:15` but never emitted from `agent.go` — TUI clients must poll rather than receiving real-time SSE updates.

### Recommendations

1. **Implement plugin-based status reporting** (primary): Write a `cortex-status.ts` OpenCode plugin that hooks `session.idle`, `tool.execute.before`, `permission.asked`, and `session.error` to POST status updates to `/agent/status`. This is symmetric with Claude Code's hook model — same push-based approach, same endpoint, same env vars (`CORTEX_TICKET_ID`, `CORTEX_PROJECT`, `CORTEX_DAEMON_URL`).

2. **Fix SessionStatus SSE emission**: Add `bus.Emit()` call in `agent.go` after `UpdateStatus()` succeeds, so TUI clients get real-time status updates for both Claude Code and OpenCode agents.

3. **Investigate plugin injection**: Determine if `OPENCODE_CONFIG_CONTENT` can define plugins inline or if Cortex needs to write the plugin file to `.opencode/plugins/` at spawn time.

4. **Consider permission model**: With `permission.asked` hook support, evaluate whether to replace `"*": "allow"` with `"*": "ask"` to give architects visibility into tool approval requests.

### Doc Created
- "OpenCode Agent Status Hooks — Research Findings" (category: findings, tags: opencode, hooks, agent-status, integration, research)