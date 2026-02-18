---
id: 90e08291-ed90-4f17-a20e-6f777ee1785d
title: Improve Claude Code hook coverage and accuracy
type: work
tags:
    - agent
    - hooks
    - api
references:
    - doc:e363a1f1-fe6d-437d-9cb9-eee631e7e0ed
    - doc:e1c47258-a082-4b06-92ca-683cc5cad8b0
    - ticket:c16eac4d-51a6-4598-86fb-4a797dc85299
created: 2026-02-13T13:24:34.239302Z
updated: 2026-02-13T13:33:13.573125Z
---
## Problem

Cortex uses only 3 of Claude Code's 14 available hooks, missing important session lifecycle and error events. Additionally, existing hooks run synchronously (no `"async": true`), adding unnecessary HTTP latency to every tool call. The `error` agent status is defined but unreachable — no hook ever sets it.

## Requirements

### New Hooks to Add

**Priority 1:**
- **`SessionStart`** → set status to `in_progress`. Eliminates the gap between spawn (`starting`) and first tool use. Payload includes `source` field (startup/resume/clear/compact).
- **`SessionEnd`** → trigger session cleanup or set `error` status depending on `reason` field (prompt_input_exit, clear, other). Makes `AgentStatusError` reachable.
- **`PostToolUseFailure`** → set `in_progress` with error context. Payload includes `tool_name`, `error` message, and `is_interrupt` boolean.

**Priority 2:**
- **`SubagentStart`** → set `in_progress` with tool context like `"Task (Explore)"`. Payload includes `agent_id`, `agent_type`.
- **`SubagentStop`** → set `in_progress` (main agent resumes). Payload includes `agent_id`, `agent_type`.

### Existing Hook Improvements

- Add `"async": true` to all hook actions in settings generation — these are fire-and-forget status updates that should not block the agent
- Expand `hookInput` struct in `hook.go` to parse additional fields from new hook payloads (`reason`, `error`, `is_interrupt`, `agent_type`, `source`)

### Status Model Fixes

- Make `AgentStatusError` reachable via `SessionEnd` (unexpected termination) and `PostToolUseFailure`
- Consider using the existing but unused `work` field in `UpdateAgentStatusRequest` to carry error context or subagent type

## References

- Findings doc: "Claude Code Hooks Audit: Complete Gap Analysis" (doc:e363a1f1)
- Findings doc: "Cortex Agent Status Integration Architecture" (doc:e1c47258)
- Related: SSE emission fix (ticket:c16eac4d)

## Acceptance Criteria

- All 5 new hooks are configured in settings generation and handled in `hook.go`
- All hooks (existing + new) use `"async": true`
- `hookInput` struct parses relevant fields from each hook's payload
- `SessionEnd` triggers cleanup or sets `error` status based on reason
- `SessionStart` transitions status from `starting` to `in_progress`
- `PostToolUseFailure` provides error context in status update
- Build, lint, and tests pass