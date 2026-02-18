---
id: 0b8e6974-1e99-4c76-ad98-e5398acb5b01
author: claude
type: review_requested
created: 2026-02-13T13:10:23.601235Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Claude Code Hooks Audit — Complete

### Key Findings

**Claude Code exposes 14 hook events. Cortex uses 3.** All 3 current mappings are accurate.

### Current Hooks (all correct)
- `PostToolUse` → `in_progress` ✅
- `Stop` → `idle` ✅  
- `PermissionRequest` → `waiting_permission` ✅

### Gaps Identified (5 recommended additions, prioritized)

**Priority 1 — High Value:**
1. **`SessionEnd`** — Fires when session terminates. Enables stale session cleanup and makes the `error` status reachable (currently dead code). Includes `reason` field (exit, crash, clear).
2. **`PostToolUseFailure`** — Fires when a tool call fails. Provides error visibility. Includes `error` message and `is_interrupt` boolean.
3. **`SessionStart`** — Fires when session begins. Eliminates the gap between `starting` status and first tool use.

**Priority 2 — Medium Value:**
4. **`SubagentStart`/`SubagentStop`** — Fire when Claude spawns/finishes subagents. Could enrich `in_progress` with "running Explore subagent" context.

**Not needed:** `PreToolUse`, `UserPromptSubmit`, `TeammateIdle`, `TaskCompleted`, `PreCompact`, `Notification` (reasons documented in findings).

### Bonus Findings
- **All hooks should use `"async": true`** — current hooks block the agent with HTTP round-trips for fire-and-forget status updates
- **`error` status is dead code** — defined in session.go but no hook ever sets it
- **`work` field in UpdateAgentStatusRequest is unused** — could carry error context
- **`SessionStatus` SSE event never emitted** — existing ticket `fix-sessionstatus-c16eac4d` tracks this

### Doc Created
Full analysis with implementation plan, payload schemas, and code change specifications in: `docs/findings/claude-code-hooks-audit`