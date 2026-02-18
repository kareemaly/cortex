---
id: c2c7828a-524d-40f6-a4a3-87d4b4e37c79
author: claude
type: done
created: 2026-02-13T13:33:13.571983Z
---
## Summary

Expanded Claude Code hook coverage from 3 to 8 events, made all hooks async, and added a `work` field for error/context threading.

### Changes (6 files, 227 insertions, 24 deletions)

**New hook events (5):**
- `SessionStart` → transitions `starting` → `in_progress`
- `SessionEnd` → `idle` (normal exit via `prompt_input_exit`/`clear`) or `error` with reason in `work`
- `PostToolUseFailure` → `in_progress` with error message in `work`
- `SubagentStart` → `in_progress` with `"Task ({agent_type})"` as tool name
- `SubagentStop` → `in_progress` with nil tool (main agent resumes)

**Async execution:** All 8 hooks now set `async: true`, eliminating HTTP latency from the agent's critical path.

**Work field:** New `Work *string` field threaded through `Session` → `Store.UpdateStatus()` → API handler → SSE event payload. Enables `AgentStatusError` to be reached (previously defined but unreachable).

### Files changed
- `internal/session/session.go` — Added `Work *string` field
- `internal/session/store.go` — Extended `UpdateStatus` signature with `work` param
- `internal/core/spawn/settings.go` — Added `Async` to `HookAction`, 5 new event fields to `HooksConfig`, updated `GenerateSettingsConfig`
- `cmd/cortexd/commands/hook.go` — 5 new commands, expanded `hookInput`, `work` in `postAgentStatus`
- `internal/daemon/api/agent.go` — Passes `work` to store and SSE events
- `internal/session/store_test.go` — Updated test calls for new signature

### Verification
- `make build` ✓
- `make lint` — 0 issues ✓
- `make test` — all pass ✓
- `git push` — pre-push hooks passed ✓

Commit: 67754db on main