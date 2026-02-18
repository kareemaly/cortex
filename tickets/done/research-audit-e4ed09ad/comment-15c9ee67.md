---
id: 15c9ee67-9677-4ba8-bfe8-9c1b0a7c5e47
author: claude
type: done
created: 2026-02-13T13:20:09.533634Z
---
## Summary

Audited Claude Code's hook system against Cortex's current implementation. Claude Code exposes 14 hook events; Cortex uses 3.

### Key Findings

1. **All 3 current hook-to-status mappings are correct** — PostToolUse→in_progress, Stop→idle, PermissionRequest→waiting_permission are semantically accurate.

2. **5 hooks recommended for addition**, prioritized:
   - **SessionEnd** (P1) — enables stale session cleanup and makes the `error` status reachable (currently dead code). Includes termination `reason` field.
   - **PostToolUseFailure** (P1) — provides error visibility with `error` message and `is_interrupt` boolean.
   - **SessionStart** (P1) — eliminates the `starting`→`in_progress` gap that exists until the first tool call.
   - **SubagentStart/SubagentStop** (P2) — enriches `in_progress` with subagent type context (Explore, Bash, Plan).

3. **6 hooks not needed**: PreToolUse, UserPromptSubmit, TeammateIdle, TaskCompleted, PreCompact, Notification — reasons documented in findings.

4. **Bonus findings**:
   - All hooks should use `"async": true` to eliminate synchronous HTTP latency on every tool call
   - `AgentStatusError` is dead code — no hook ever sets it
   - `work` field in UpdateAgentStatusRequest is accepted but never stored
   - `hookInput` struct only reads `tool_name` — needs expansion for new hook payloads

### Artifacts
- Doc created: `docs/findings/claude-code-hooks-audit` — full gap analysis with implementation plan, payload schemas, and code change specifications

### Recommended Next Steps
1. **Quick win**: Add `"async": true` to all existing hook actions in `settings.go` (one-line change per hook, immediate latency improvement)
2. **Phase 1 ticket**: Add SessionEnd + SessionStart hooks (session lifecycle)
3. **Phase 2 ticket**: Add PostToolUseFailure hook (error visibility)
4. **Phase 3 ticket**: Add SubagentStart/SubagentStop hooks (status enrichment)
5. **Related**: Fix SessionStatus SSE emission (existing ticket `fix-sessionstatus-c16eac4d`) to get real-time status pushes to TUI