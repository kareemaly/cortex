# Agent Status Tracking

## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

No visibility into what agents are currently doing. Kanban shows tickets but not agent activity (thinking, running tool, idle, waiting for permission).

## Requirements

### 1. Generate settings.json per spawn

When spawning agents, generate a settings.json alongside mcp-config.json:

```json
{
  "hooks": {
    "PostToolUse": [{
      "matcher": "*",
      "hooks": [{"type": "command", "command": "cortexd hook post-tool-use"}]
    }],
    "Stop": [{
      "hooks": [{"type": "command", "command": "cortexd hook stop"}]
    }],
    "PermissionRequest": [{
      "hooks": [{"type": "command", "command": "cortexd hook permission-request"}]
    }]
  }
}
```

### 2. Update spawn command

Add `--settings <path>` to claude command:

```bash
CORTEX_TICKET_ID=<id> CORTEX_PROJECT=<path> claude '<prompt>' --mcp-config <path> --settings <path>
```

### 3. Add `cortexd hook` subcommand

Handles hook events from claude:

```bash
cortexd hook post-tool-use   # reads JSON from stdin
cortexd hook stop
cortexd hook permission-request
```

Each command:
- Reads `CORTEX_TICKET_ID` and `CORTEX_PROJECT` from env
- Reads hook JSON from stdin (claude provides tool info)
- POSTs to daemon API

### 4. Add API endpoint

```
POST /agent/status
X-Cortex-Project: /path/to/project

{
  "ticket_id": "abc123",
  "status": "in_progress",
  "tool": "Bash",
  "work": "Running tests..."
}
```

Updates `CurrentStatus` and appends to `StatusHistory` on the ticket's session.

### 5. Display in kanban

Show current status on ticket cards (e.g., "thinking...", "Bash: running tests").

## Implementation

### Commits Pushed
- `32d7152` feat: add agent status tracking via Claude hooks
- `6578828` Merge branch 'ticket/2026-01-22-agent-status-tracking'

### Key Files Changed

**Created:**
- `internal/core/spawn/settings.go` - Generates Claude settings.json with hooks config
- `internal/daemon/api/agent.go` - POST /agent/status API handler
- `cmd/cortexd/commands/hook.go` - Hook subcommands (post-tool-use, stop, permission-request)

**Modified:**
- `internal/core/spawn/command.go` - Added SettingsPath field and --settings flag
- `internal/core/spawn/spawn.go` - Integrated settings generation, added CORTEX_PROJECT env var
- `internal/daemon/api/types.go` - Added AgentStatus/AgentTool to TicketSummary
- `internal/daemon/api/server.go` - Added /agent/status route
- `internal/cli/sdk/client.go` - Added AgentStatus/AgentTool fields
- `internal/cli/tui/kanban/column.go` - Added formatAgentStatus() with status symbols

### Important Decisions
- Hook commands fail gracefully (exit 0) if env vars missing or daemon unreachable
- Status symbols: ▶ starting, ● in_progress, ○ idle, ⏸ waiting_permission, ✗ error
- Tool names truncated to 8 chars in kanban display
- CORTEX_PROJECT env var added alongside CORTEX_TICKET_ID for hooks

### Scope Changes
- None - implemented as specified
