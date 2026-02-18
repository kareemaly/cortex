---
id: e1c47258-a082-4b06-92ca-683cc5cad8b0
title: Cortex Agent Status Integration Architecture
tags:
    - architecture
    - agent-status
    - hooks
    - claude-code
    - opencode
    - permissions
created: 2026-02-13T10:27:35.767522Z
updated: 2026-02-13T10:27:35.767522Z
---
# Cortex Agent Status Integration Architecture

## Executive Summary

Cortex integrates with Claude Code and OpenCode agents through a comprehensive hook-based system that detects agent state changes (idle, working, waiting for permissions) and tracks them in a centralized session store. The architecture uses Claude's settings hook callbacks to update agent status in real-time, enabling the companion pane and UIs to reflect agent state without active polling of the tmux window.

## Core Architecture

### 1. Session State Model

**Location**: `internal/session/session.go`

Agent status is represented by five discrete states:

```go
const (
    AgentStatusStarting          AgentStatus = "starting"
    AgentStatusInProgress        AgentStatus = "in_progress"
    AgentStatusIdle              AgentStatus = "idle"
    AgentStatusWaitingPermission AgentStatus = "waiting_permission"
    AgentStatusError             AgentStatus = "error"
)
```

Sessions are stored per-project in `.cortex/sessions.json`:

```go
type Session struct {
    Type          SessionType `json:"type"`
    TicketID      string      `json:"ticket_id,omitempty"`
    Agent         string      `json:"agent"`
    TmuxWindow    string      `json:"tmux_window"`
    WorktreePath  *string     `json:"worktree_path,omitempty"`
    FeatureBranch *string     `json:"feature_branch,omitempty"`
    StartedAt     time.Time   `json:"started_at"`
    Status        AgentStatus `json:"status"`
    Tool          *string     `json:"tool,omitempty"`  // Currently used tool during in_progress
}
```

### 2. Spawn Orchestration & State Detection

**Location**: `internal/core/spawn/orchestrate.go`, `internal/core/spawn/state.go`

Cortex tracks three **session states** (normal/active/orphaned) independent of agent status:

```go
const (
    StateNormal   SessionState = "normal"    // No session record exists
    StateActive   SessionState = "active"    // Session exists and tmux window is running
    StateOrphaned SessionState = "orphaned"  // Session exists but tmux window closed
)
```

State detection uses:
- **Session store** (`.cortex/sessions.json`): Persistent session record
- **Tmux manager**: `WindowExists(session, windowName)` to check if window still runs
- **Mode matrix**: Defines valid transitions (normal/resume/fresh spawn modes)

```
State/Mode Matrix:
| Mode    | Normal      | Active         | Orphaned    |
|---------|-------------|----------------|-------------|
| normal  | Spawn new   | AlreadyActive  | StateError  |
| resume  | StateError  | StateError     | Resume      |
| fresh   | StateError  | StateError     | Fresh       |
```

### 3. Hook-Based Status Updates

**Location**: `cmd/cortexd/commands/hook.go`, `internal/core/spawn/settings.go`

Claude Code (not OpenCode) is configured with hooks in a generated `settings.json` file that triggers cortexd commands on agent state changes:

```json
{
  "hooks": {
    "PostToolUse": [{
      "matcher": "*",
      "hooks": [{
        "type": "command",
        "command": "/path/to/cortexd hook post-tool-use"
      }]
    }],
    "Stop": [{
      "hooks": [{
        "type": "command",
        "command": "/path/to/cortexd hook stop"
      }]
    }],
    "PermissionRequest": [{
      "hooks": [{
        "type": "command",
        "command": "/path/to/cortexd hook permission-request"
      }]
    }]
  }
}
```

**Hook handlers** (3 commands in `cmd/cortexd/commands/hook.go`):

- **post-tool-use**: Called after agent uses a tool → updates status to `in_progress` with tool name
- **stop**: Called when agent stops thinking → updates status to `idle`
- **permission-request**: Called when agent needs permission → updates status to `waiting_permission`

All hooks read environment variables to identify the ticket:
- `CORTEX_TICKET_ID`: Ticket ID or "architect" for architect sessions
- `CORTEX_PROJECT`: Project path
- `CORTEX_DAEMON_URL`: Daemon HTTP URL (defaults to `http://127.0.0.1:4200`)

They POST to `POST /agent/status` endpoint with JSON payload:

```json
{
  "ticket_id": "abc123...",
  "status": "in_progress|idle|waiting_permission",
  "tool": "tool_name_optional"
}
```

### 4. Environment Setup During Spawn

**Location**: `internal/core/spawn/spawn.go`, `internal/core/spawn/launcher.go`

When spawning any session (ticket, architect, meta), Cortex:

1. **Generates settings config** (Claude only, not OpenCode):
   - Creates `cortex-settings-{identifier}.json` with hooks
   - Passed to Claude via `--settings` flag

2. **Generates MCP config**:
   - Creates `cortex-mcp-{identifier}.json` with ticket/project context
   - Passed via `--mcp-config` flag

3. **Sets environment variables** in launcher script:
   ```bash
   export CORTEX_TICKET_ID={ticketID}
   export CORTEX_PROJECT={projectPath}
   export CORTEX_DAEMON_URL={daemonURL}
   ```

4. **For OpenCode only**: Generates `OPENCODE_CONFIG_CONTENT` JSON env var since OpenCode doesn't support hooks

5. **Creates launcher script** (`cortex-launcher-{identifier}.sh`):
   - Bash script that exports env vars and runs agent command
   - Sets up cleanup trap to remove temp files on exit
   - Launched in tmux left pane (agent)

### 5. Agent Status Persistence & Updates

**Location**: `internal/daemon/api/agent.go`, `internal/session/store.go`

The HTTP endpoint `POST /agent/status` is the single point where agent status is updated:

```go
type UpdateAgentStatusRequest struct {
    TicketID string  `json:"ticket_id"`
    Status   string  `json:"status"`
    Tool     *string `json:"tool,omitempty"`
    Work     *string `json:"work,omitempty"`
}
```

Flow:
1. Hook command POSTs to `/agent/status` endpoint
2. Handler validates status is one of the 5 allowed states
3. Calls `sessStore.UpdateStatus(ticketShortID, agentStatus, tool)`
4. Session store locks, reads `.cortex/sessions.json`, updates status field, writes back
5. **NOTE**: No events are currently emitted on status updates (opportunity for improvement)

### 6. Event Bus (Currently Unused for Status)

**Location**: `internal/events/bus.go`

Event types defined:

```go
const (
    TicketCreated   EventType = "ticket_created"
    TicketUpdated   EventType = "ticket_updated"
    TicketMoved     EventType = "ticket_moved"
    SessionStarted  EventType = "session_started"
    SessionEnded    EventType = "session_ended"
    SessionStatus   EventType = "session_status"  // Defined but not emitted
    CommentAdded    EventType = "comment_added"
    ReviewRequested EventType = "review_requested"
    // ...
)
```

Events are emitted via in-process pub/sub bus and streamed to clients via SSE endpoint (`GET /events`). However, agent status updates do NOT currently trigger SessionStatus events (architectural gap).

## Claude Code vs OpenCode Integration

### Claude Code Configuration

**Default**: `internal/install/defaults/claude-code/cortex.yaml`

```yaml
architect:
  agent: claude
  args:
    - "--allow-dangerously-skip-permissions"
    - "--allowedTools"
    - "mcp__cortex__listTickets,mcp__cortex__readTicket"
ticket:
  work:
    agent: claude
    args:
      - "--permission-mode"
      - "plan"
      - "--allow-dangerously-skip-permissions"
      - "--allowedTools"
      - "mcp__cortex__readReference"
```

**Integration points**:
- Hooks: ✅ Supported via settings.json
- Status tracking: ✅ Full support (in_progress, idle, waiting_permission)
- MCP tools: ✅ Receives cortex MCP server on stdio
- Permissions: ✅ `--permission-mode plan` + hooks track waiting_permission state
- Tool tracking: ✅ Tool name captured in `Post-ToolUse` hook

### OpenCode Configuration

**Default**: `internal/install/defaults/opencode/cortex.yaml`

```yaml
architect:
  agent: opencode
ticket:
  work:
    agent: opencode
```

**Integration points**:
- Hooks: ❌ NOT supported (OpenCode doesn't have hook system)
- Status tracking: ⚠️ Limited (no way to detect waiting_permission)
- MCP config: ✅ Via `OPENCODE_CONFIG_CONTENT` env var (JSON passed directly)
- Agent config: ✅ Sent in env var with `"permission": {"*": "allow"}` for full bypass

**Key difference**: OpenCode receives agent configuration (MCP servers, system prompt, permissions) entirely through the `OPENCODE_CONFIG_CONTENT` environment variable. No hooks means no real-time status updates.

```go
// GenerateOpenCodeConfigContent (internal/core/spawn/opencode_config.go)
type OpenCodeAgentConfig struct {
    Description string            `json:"description"`
    Mode        string            `json:"mode"` // e.g., "bypassPermissions"
    Prompt      string            `json:"prompt"`
    Permission  map[string]string `json:"permission"` // {"*": "allow"}
}
```

## Spawn Execution Flow (Detailed)

**Location**: `internal/core/spawn/spawn.go`

### 1. Validation & State Detection
- Validate spawn request
- Check if ticket already has active session
- Detect current session state (normal/active/orphaned)

### 2. Session Creation
- Create session record in `.cortex/sessions.json`
- Set initial status to `AgentStatusStarting`

### 3. Configuration Generation
- **MCP Config**: `internal/core/spawn/command.go` → `GenerateMCPConfig()`
- **Settings Config** (Claude only): `internal/core/spawn/settings.go` → `GenerateSettingsConfig()`
- **OpenCode Config** (if OpenCode agent): `internal/core/spawn/opencode_config.go` → `GenerateOpenCodeConfigContent()`

### 4. Prompt Loading & Building
- Resolve prompt files from:
  - Project directory: `.cortex/prompts/{role}/{ticketType}/{KICKOFF,SYSTEM,APPROVE}.md`
  - Base config (extend): fallback if project doesn't have custom prompts
  - Defaults: embedded in cortex binary
- Build system prompt + kickoff prompt
- Write temp files (cleaned up on exit via trap)

### 5. Launcher Script Generation
- Build command based on agent type:
  - **Claude**: `claude "$(cat prompt.txt)" --mcp-config X --settings X --append-system-prompt "$(cat sysprompt.txt)"`
  - **OpenCode**: `opencode --agent cortex --prompt "$(cat prompt.txt)"`
- Set environment variables (CORTEX_TICKET_ID, CORTEX_PROJECT, etc.)
- Write bash script with trap cleanup

### 6. Tmux Spawn
- `SpawnAgent()` or `SpawnArchitect()` in tmux manager
- Creates tmux session (if needed)
- Creates window with agent pane (left)
- If companion command provided: splits horizontally, creates companion pane (right)
- Runs launcher script in agent pane

### 7. Return Result
Returns `SpawnResult` with window info and paths

## Tmux Window Layout

**Ticket/Meta Session** (via `SpawnAgent`):
```
tmux session:
  window N (agent-ticket-id):
    pane 0 (left, 30%):   [agent - running launcher script]
    pane 1 (right, 70%):  [companion - e.g., file monitor, logs]
```

**Architect Session** (via `SpawnArchitect`):
```
tmux session:
  window 0 (architect):
    pane 0 (left, 30%):   [architect agent]
    pane 1 (right, 70%):  [companion - e.g., ticket list, logs]
```

## MCP Server Integration

**Location**: `internal/daemon/mcp/server.go`, `cmd/cortexd/commands/mcp.go`

When agent starts, cortexd launches MCP server on stdio transport:

```bash
cortexd mcp --ticket-id=abc123 --ticket-type=work
```

MCP server:
- Reads config from environment variables
- Contacts daemon via HTTP API (for mutation operations)
- Exposes tools based on session type:
  - **Ticket sessions**: `readReference`, `addComment`, `addBlocker`, `requestReview`, `concludeSession`, `createDoc` (research only)
  - **Architect sessions**: Full tool set for ticket/doc management, spawning
  - **Meta sessions**: Project management, global config access

## OpenCode Specific Implementation

### No Hook Support

Since OpenCode doesn't have hooks, status changes cannot be detected in real-time. The architecture handles this by:

1. **Permission bypass**: Config includes `"permission": {"*": "allow"}` to automatically approve all operations
2. **No waiting_permission state**: OpenCode agent won't be tracked as waiting for permission
3. **Only starting status**: Session initial status set to `AgentStatusStarting`, never transitions

### Config via Environment Variable

```bash
export OPENCODE_CONFIG_CONTENT='{"agent":{"cortex":{"description":"...","mode":"bypassPermissions","prompt":"...","permission":{"*":"allow"}}},"mcp":{...}}'
opencode --agent cortex --prompt "$(cat prompt.txt)"
```

### MCP Configuration Format

OpenCode config differs from Claude's MCP config structure:
- Uses direct JSON structure instead of Claude's nested format
- MCP servers defined as: `"type": "local"`, `"command": [...]`, `"environment": {...}`

## Client-Side State Polling (Workaround)

Since status updates don't trigger events, clients use polling patterns:

**Kanban TUI** (not shown in code, but likely pattern):
- Periodically calls `GET /sessions` to list all active sessions
- Reads `Status` and `Tool` fields from session response
- Displays status badges in session list

**API Response** includes session state:
```go
type TicketSummary struct {
    // ...
    AgentStatus      *string    `json:"agent_status,omitempty"`
    AgentTool        *string    `json:"agent_tool,omitempty"`
    HasActiveSession bool       `json:"has_active_session"`
    IsOrphaned       bool       `json:"is_orphaned,omitempty"`
    SessionStartedAt *time.Time `json:"session_started_at,omitempty"`
}
```

## Lifecycle Hooks (Different from Status Hooks)

**Location**: `internal/lifecycle/` (mentioned in CLAUDE.md, not implemented in core)

Defined in `.cortex/cortex.yaml` under `lifecycle` section. Run on:
- **pickup**: When ticket moved to progress
- **review**: When requesting review
- **approve**: When approving ticket

Support template variables like `{{.Slug}}`, `{{.CommitMessage}}`, etc.

## Key Integration Points for New Agents

### For Claude-Like Agents (with Hook Support)

1. **In settings generation**: Ensure all three hook types are configured
2. **In launcher**: Pass `--settings` flag with generated config
3. **Environment variables**: Set CORTEX_TICKET_ID, CORTEX_PROJECT, CORTEX_DAEMON_URL
4. **Status endpoint**: Ensure agent can HTTP POST to daemon (or wrapper script can)

### For OpenCode-Like Agents (No Hook Support)

1. **Skip settings generation**: Don't generate settings.json or pass `--settings`
2. **Config via env var**: Pass all configuration via OPENCODE_CONFIG_CONTENT
3. **Permission mode**: Set appropriate permission bypass/mode in config
4. **No status updates**: Accept that agent status won't transition beyond "starting"
5. **Ensure MCP works**: Validate that MCP server communicates correctly

## Known Gaps & Opportunities

### Current Gaps

1. **No event emission on status change**: SessionStatus events are defined but never emitted
   - Workaround: Clients poll `GET /sessions`
   - Solution: Add `h.deps.Bus.Emit(Event{...SessionStatus...})` in agent.go UpdateStatus

2. **No OpenCode status tracking**: OpenCode agents don't report waiting_permission
   - Fundamental limitation: OpenCode has no hook system
   - Alternative: Monitor OpenCode logs or stdout for permission prompts (not implemented)

3. **No companion pane monitoring**: Companion pane (right) is never monitored
   - Could track companion process separately for orphan detection
   - Currently only left pane (agent) checked via `WindowExists()`

4. **Limited pane readiness detection**: Only checks if tmux window exists, not if agent is ready
   - Could poll agent pane output or use additional hooks for initialization complete
   - Currently just waits for spawn to complete (no wait loop)

### Enhancement Opportunities

1. **Real-time status events**: Emit SessionStatus events on hook callbacks
2. **OpenCode monitoring**: Implement alternative status tracking for OpenCode (log parsing, stdout monitoring)
3. **Agent readiness**: Track when agent first becomes idle (not just when spawned)
4. **Tool execution time**: Track duration of tool usage (captured in session, could be used for analytics)
5. **Error status**: Propagate agent errors (parse stderr) to waiting_permission state

## Summary Table

| Aspect | Claude Code | OpenCode | Notes |
|--------|-------------|----------|-------|
| Status Updates | Hooks (3 types) | None | OpenCode needs alternative approach |
| Permission Detection | waiting_permission | No | OpenCode bypasses permissions |
| MCP Transport | Stdio | Stdio | Both use stdio |
| Configuration | --settings + --mcp-config | OPENCODE_CONFIG_CONTENT | Different config methods |
| Session Tracking | Full (5 states) | Minimal (starting only) | Status transitions limited for OpenCode |
| Event Emission | Defined but unused | N/A | Opportunity to improve |
| Spawn Modes | normal/resume/fresh | normal/resume/fresh | Same logic applies |
| Worktree Support | Yes | Yes | Both support git worktrees |

## Files Reference

**Core Architecture**:
- `internal/session/session.go` - AgentStatus definitions
- `internal/core/spawn/orchestrate.go` - State detection & spawn flow
- `internal/core/spawn/state.go` - StateInfo, state detection helpers

**Hook System**:
- `cmd/cortexd/commands/hook.go` - Hook handlers (post-tool-use, stop, permission-request)
- `internal/core/spawn/settings.go` - Settings config generation

**Configuration**:
- `internal/core/spawn/launcher.go` - Launcher script generation
- `internal/core/spawn/opencode_config.go` - OpenCode config structure
- `internal/install/defaults/` - Default configurations for claude-code and opencode

**API**:
- `internal/daemon/api/agent.go` - UpdateStatus endpoint
- `internal/daemon/api/sessions.go` - Session listing & management
- `internal/daemon/api/architect.go` - Architect spawn & state

**MCP**:
- `internal/daemon/mcp/server.go` - MCP server initialization
- `internal/daemon/mcp/tools_ticket.go` - Ticket tools
- `cmd/cortexd/commands/mcp.go` - MCP command entry point

**Events**:
- `internal/events/bus.go` - Event bus (unused for status)
- `internal/daemon/api/events.go` - SSE streaming (would benefit from status events)

**Tmux**:
- `internal/tmux/command.go` - SpawnAgent, SpawnArchitect commands
- `internal/tmux/client.go` - Client attachment detection (IsUserAttached)
