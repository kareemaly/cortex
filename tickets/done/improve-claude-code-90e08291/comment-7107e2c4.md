---
id: 7107e2c4-bd96-4a1a-87eb-b57976e0e8ac
author: claude
type: review_requested
created: 2026-02-13T13:32:17.806465Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Improved Claude Code hook coverage and accuracy by adding 5 new hook events, making all hooks async, and threading error/subagent context through a new `work` field.

### Changes

**`internal/session/session.go`** — Added `Work *string` field to `Session` struct for error messages and contextual info.

**`internal/session/store.go`** — Extended `UpdateStatus(ticketShortID, status, tool, work)` to accept and persist the `work` field.

**`internal/core/spawn/settings.go`** — 
- Added `Async bool` field to `HookAction` struct
- Added 5 new hook event fields to `HooksConfig`: `PostToolUseFailure`, `SessionStart`, `SessionEnd`, `SubagentStart`, `SubagentStop`
- Updated `GenerateSettingsConfig()` to emit all 8 hooks with `Async: true`

**`cmd/cortexd/commands/hook.go`** —
- Expanded `hookInput` struct with fields: `Reason`, `Error`, `IsInterrupt`, `AgentType`, `AgentID`, `Source`
- Updated `postAgentStatus()` to accept and send `work` parameter
- Added 5 new hook commands:
  - `session-start` → transitions `starting` → `in_progress`
  - `session-end` → `idle` (normal exit) or `error` with reason in `work` (abnormal)
  - `post-tool-use-failure` → `in_progress` with error message in `work`
  - `subagent-start` → `in_progress` with `"Task ({agent_type})"` as tool
  - `subagent-stop` → `in_progress` with nil tool (main agent resumes)

**`internal/daemon/api/agent.go`** — Passes `req.Work` through to `sessStore.UpdateStatus()` and includes `work` in SSE event payloads.

**`internal/session/store_test.go`** — Updated all `UpdateStatus` calls to match new 4-arg signature.

### Verification
- `make build` ✓
- `make lint` — 0 issues ✓
- `make test` — all pass, no failures ✓