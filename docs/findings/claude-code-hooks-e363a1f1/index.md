---
id: e363a1f1-fe6d-437d-9cb9-eee631e7e0ed
title: 'Claude Code Hooks Audit: Complete Gap Analysis'
tags:
    - hooks
    - claude-code
    - agent-status
    - audit
created: 2026-02-13T13:10:09.150197Z
updated: 2026-02-13T13:10:09.150197Z
---
## Executive Summary

Claude Code exposes **14 hook events**. Cortex uses **3**. Several of the unused hooks would meaningfully improve agent status tracking fidelity — particularly around session lifecycle, error detection, and subagent awareness.

---

## Current Implementation

### Hooks in Use (3 of 14)

| Hook Event | Cortex Status | Handler | Notes |
|---|---|---|---|
| `PostToolUse` | `in_progress` | `cortexd hook post-tool-use` | Reads `tool_name` from stdin JSON. Matcher: `*` |
| `Stop` | `idle` | `cortexd hook stop` | No stdin parsing |
| `PermissionRequest` | `waiting_permission` | `cortexd hook permission-request` | No stdin parsing |

**Implementation files:**
- Hook handlers: `cmd/cortexd/commands/hook.go`
- Settings generation: `internal/core/spawn/settings.go`
- Status API endpoint: `internal/daemon/api/agent.go`
- Session model: `internal/session/session.go`

### Defined Agent Statuses (5)

| Status | Source | Notes |
|---|---|---|
| `starting` | Set on session creation (spawn) | Never updated by hooks — set programmatically |
| `in_progress` | `PostToolUse` hook | Working, with optional tool name |
| `idle` | `Stop` hook | Waiting for input |
| `waiting_permission` | `PermissionRequest` hook | Blocked on user approval |
| `error` | Defined but **never set by any hook** | Dead code status |

---

## Complete Claude Code Hook Events (14)

### Full Inventory

| # | Event | Fires When | Can Block? | Used by Cortex? |
|---|---|---|---|---|
| 1 | **SessionStart** | Session begins or resumes | No | **No** |
| 2 | **UserPromptSubmit** | User submits a prompt | Yes | **No** |
| 3 | **PreToolUse** | Before tool execution | Yes | **No** |
| 4 | **PermissionRequest** | Permission dialog appears | Yes | **Yes** → `waiting_permission` |
| 5 | **PostToolUse** | After tool succeeds | No | **Yes** → `in_progress` |
| 6 | **PostToolUseFailure** | After tool fails | No | **No** |
| 7 | **Notification** | Claude Code sends a notification | No | **No** |
| 8 | **SubagentStart** | Subagent spawned | No | **No** |
| 9 | **SubagentStop** | Subagent finishes | Yes | **No** |
| 10 | **Stop** | Claude finishes responding | Yes | **Yes** → `idle` |
| 11 | **TeammateIdle** | Agent team teammate going idle | Yes | **No** |
| 12 | **TaskCompleted** | Task marked completed | Yes | **No** |
| 13 | **PreCompact** | Before context compaction | No | **No** |
| 14 | **SessionEnd** | Session terminates | No | **No** |

### Hook Input Payloads (Key Fields)

**Common fields (all hooks):**
```json
{
  "session_id": "abc123",
  "transcript_path": "/path/to/transcript.jsonl",
  "cwd": "/working/directory",
  "permission_mode": "default|plan|acceptEdits|dontAsk|bypassPermissions",
  "hook_event_name": "EventName"
}
```

**Event-specific payloads of interest:**

| Event | Key Fields |
|---|---|
| `SessionStart` | `source` (startup/resume/clear/compact), `model`, `agent_type` |
| `PostToolUse` | `tool_name`, `tool_input`, `tool_response` |
| `PostToolUseFailure` | `tool_name`, `tool_input`, `error`, `is_interrupt` |
| `Notification` | `message`, `title`, `notification_type` (permission_prompt, idle_prompt, auth_success, elicitation_dialog) |
| `SubagentStart` | `agent_id`, `agent_type` |
| `SubagentStop` | `agent_id`, `agent_type`, `agent_transcript_path` |
| `Stop` | `stop_hook_active` (boolean) |
| `SessionEnd` | `reason` (clear/logout/prompt_input_exit/bypass_permissions_disabled/other) |
| `PreCompact` | `trigger` (manual/auto) |

---

## Accuracy Assessment of Current Mappings

### PostToolUse → `in_progress` ✅ Correct
The agent just used a tool, so it's actively working. Tool name is captured from stdin JSON. This is the right mapping.

### Stop → `idle` ✅ Mostly Correct (with caveat)
`Stop` fires when Claude finishes its turn — it has produced its final response and is waiting for the next user message. Mapping this to `idle` is semantically accurate for Cortex's purposes.

**Caveat**: If the agent has genuinely completed its work (called `concludeSession` MCP tool), the final `Stop` event will briefly set status to `idle` before the session cleanup removes it. This is a harmless race condition since session teardown follows immediately.

### PermissionRequest → `waiting_permission` ✅ Correct
The permission dialog is visible and the agent is blocked until the user approves or denies. Correct mapping.

---

## Gap Analysis: Recommended New Hooks

### Priority 1 — High Value, Easy Implementation

#### 1. `SessionEnd` → session cleanup / `error` detection
**Why:** When a session terminates unexpectedly (user kills it, crashes, etc.), the status stays stale at whatever the last hook reported. The `SessionEnd` hook fires on termination and includes a `reason` field.

**Mapping logic:**
- `reason: "prompt_input_exit"` → Clean exit, trigger session cleanup (equivalent to user quitting)
- `reason: "other"` or unexpected → Could set `error` status or trigger cleanup
- `reason: "clear"` → User cleared session, cleanup

**Impact:** Would finally make the `error` status reachable and enable automatic stale session detection without relying solely on tmux polling.

#### 2. `PostToolUseFailure` → `error` or enriched `in_progress`
**Why:** Currently, if a tool call fails (e.g., bash command returns non-zero, file not found), Cortex has no visibility. The agent is still working, but something went wrong.

**Mapping options:**
- Option A: Set `in_progress` with a `tool` field showing `"Edit (failed)"` or similar — keeps it simple
- Option B: Set a new sub-status or use the `error` status — more disruptive but more informative
- Option C: Set `in_progress` and log the error via the `work` field (already in the request struct but unused)

**Payload available:** `tool_name`, `error` message, `is_interrupt` boolean

**Impact:** Better debugging visibility. The `is_interrupt` field could distinguish user-interrupted tools from genuine failures.

#### 3. `Notification` → improved `waiting_permission` / `idle` detection
**Why:** The `Notification` hook carries a `notification_type` field with values:
- `permission_prompt` — permission dialog shown (redundant with `PermissionRequest` but confirms state)
- `idle_prompt` — agent is idle and prompting user (stronger idle signal)

**Mapping:** Could use as confirmation/deduplication, or add `idle_prompt` as a distinct signal. Not critical if `Stop` and `PermissionRequest` are already covering these states. Lower priority than the others.

### Priority 2 — Medium Value, Enriches Status

#### 4. `SubagentStart` / `SubagentStop` → enriched `in_progress`
**Why:** When Claude spawns a Task subagent (e.g., `Explore`, `Bash`, `Plan`), the main agent is waiting for the subagent to finish. Currently this looks like `in_progress` with whatever the last tool was.

**Mapping:**
- `SubagentStart` → `in_progress` with tool = `"Task (Explore)"` or `"subagent:{agent_type}"`
- `SubagentStop` → `in_progress` (agent resumes main work)

**Payload available:** `agent_id`, `agent_type` (Bash, Explore, Plan, etc.)

**Impact:** TUI could show "spawning Explore subagent" instead of a generic "in_progress". Moderate UX improvement.

#### 5. `SessionStart` → confirm session alive
**Why:** Fires when the session begins or resumes. Could be used to transition from `starting` to `in_progress` immediately rather than waiting for the first tool use.

**Matchers available:** `startup`, `resume`, `clear`, `compact`

**Mapping:**
- `startup` → `in_progress` (agent has loaded and is beginning work)
- `resume` → `in_progress` (agent resumed from interrupted session)
- `compact` → could briefly show "compacting context" but this is low value

**Impact:** Eliminates the gap between spawn (status = `starting`) and first tool use (status = `in_progress`). The agent may spend several seconds "thinking" before its first tool call.

### Priority 3 — Low Value / Not Applicable

#### 6. `PreToolUse` — Not needed for status tracking
This is primarily for blocking/security (preventing dangerous commands). No status tracking value since `PostToolUse` already covers the "working" state. Would add latency with no benefit.

#### 7. `UserPromptSubmit` — Not applicable
In Cortex's architecture, the "user" messages to agents come from MCP tools or the system prompt, not from interactive user input. This hook is irrelevant for automated agent workflows.

#### 8. `TeammateIdle` / `TaskCompleted` — Not applicable (yet)
These are for Claude Code's agent teams feature. Not relevant unless Cortex adopts multi-agent teams within a single Claude session. Could become relevant in the future.

#### 9. `PreCompact` — Marginal value
Could show a brief "compacting" status, but compaction is fast and the status would flash too quickly to be useful. Not worth the implementation effort.

---

## Recommended Implementation Plan

### Phase 1: Session Lifecycle (Highest ROI)
1. **Add `SessionEnd` hook** — enables stale session cleanup and `error` status
2. **Add `SessionStart` hook** — eliminates `starting` → `in_progress` gap

### Phase 2: Error Visibility
3. **Add `PostToolUseFailure` hook** — enables error tracking and the `error` status

### Phase 3: Status Enrichment (Optional)
4. **Add `SubagentStart`/`SubagentStop` hooks** — enriches `in_progress` with subagent context

### Settings Config Changes Required

The `HooksConfig` struct in `internal/core/spawn/settings.go` would need new fields:

```go
type HooksConfig struct {
    PostToolUse        []HookEntry `json:"PostToolUse,omitempty"`
    PostToolUseFailure []HookEntry `json:"PostToolUseFailure,omitempty"`
    Stop               []HookEntry `json:"Stop,omitempty"`
    PermissionRequest  []HookEntry `json:"PermissionRequest,omitempty"`
    SessionStart       []HookEntry `json:"SessionStart,omitempty"`
    SessionEnd         []HookEntry `json:"SessionEnd,omitempty"`
    SubagentStart      []HookEntry `json:"SubagentStart,omitempty"`
    SubagentStop       []HookEntry `json:"SubagentStop,omitempty"`
}
```

New hook commands in `cmd/cortexd/commands/hook.go`:
- `cortexd hook session-start`
- `cortexd hook session-end`
- `cortexd hook post-tool-use-failure`
- `cortexd hook subagent-start`
- `cortexd hook subagent-stop`

### Async Hook Consideration

All Cortex hooks are fire-and-forget status updates — they don't need to block the agent. Claude Code supports `"async": true` for command hooks, which runs them in the background without blocking the agent's execution. **All Cortex hooks should use `"async": true`** since:
- Status updates are non-blocking by design (5s timeout, fail-gracefully)
- Adding latency to every tool call is undesirable
- The current hooks don't use async, which means every PostToolUse adds HTTP round-trip latency

---

## Additional Findings

### 1. `SessionStatus` SSE Event Gap (Pre-existing)
The event bus defines `SessionStatus` event type (`internal/events/bus.go:15`) but `agent.go:UpdateStatus()` never emits it. There's already a ticket for this: `fix-sessionstatus-c16eac4d`. This is orthogonal to the hooks audit but would compound the value of new hooks — real-time SSE pushes for all status changes.

### 2. `error` Status is Dead Code
The `AgentStatusError` is defined (`internal/session/session.go:13`) and validated (`internal/daemon/api/agent.go:117`) but no hook ever sets it. Adding `PostToolUseFailure` and `SessionEnd` hooks would make this status reachable.

### 3. `work` Field is Unused
The `UpdateAgentStatusRequest` struct has a `Work *string` field (`agent.go:27`) that is accepted but never used or stored. This could be repurposed for richer status context (e.g., error messages from `PostToolUseFailure`).

### 4. Hook Input Struct is Minimal
The `hookInput` struct in `hook.go` only reads `tool_name`. New hooks would need to parse additional fields (`reason` for SessionEnd, `error` for PostToolUseFailure, `agent_type` for SubagentStart/Stop). Consider expanding to a more generic input struct.

### 5. Hooks Should Use Async Mode
Current hooks run synchronously, adding HTTP round-trip latency to every tool call. Since all hooks are fire-and-forget status updates, setting `"async": true` in the HookAction would eliminate this overhead without any functional change.