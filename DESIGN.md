# Cortex v1 Design

## Overview

Orchestration layer for AI coding workflows. File-based ticket management with MCP tools for agent interaction.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         cortexd (daemon)                        │
├─────────────────────────────────────────────────────────────────┤
│  HTTP API          │  MCP Server         │  Core                │
│  - /tickets/*      │  - Architect tools  │  - Ticket CRUD       │
│  - /sessions/*     │  - Ticket tools     │  - Git repo discovery│
│  - /health         │                     │  - Lifecycle hooks   │
│                    │                     │  - Tmux management   │
└─────────────────────────────────────────────────────────────────┘

Global: ~/.cortex/
├── settings.yaml           # Global config
├── daemon.log              # Daemon logs

Project: <project>/.cortex/
├── cortex.yaml             # Project config + lifecycle hooks
└── tickets/
    ├── backlog/            # Not started
    ├── progress/           # Agent working
    └── done/               # Approved/completed
```

## Global Configuration

```yaml
# ~/.cortex/settings.yaml
port: 4200
log_level: info
status_history_limit: 10    # Status history entries per session
git_diff_tool: diff         # diff | lazygit
```

## CLI Commands

```
cortex                      # Show help/usage
cortex kanban               # Open kanban TUI for current project
cortex architect            # Start/attach architect session
cortex session <id>         # Open session view TUI (read-only)
cortex spawn <ticket-id>    # Spawn ticket session
cortex list [--status=...]  # List tickets
cortex version              # Version info
cortex init                 # Initialize project
```

## Session Types

### Architect Session (Singleton)

One per project. For planning, ticket management, and orchestration.

**Spawned by:** `cortex architect` (attaches if running, spawns if not)
**MCP Tools:** Full ticket CRUD, spawn sessions, view status

### Ticket Session

One per ticket at a time. Agent implements the ticket.

**Spawned by:** Architect via MCP tool, or `cortex spawn <id>`
**Initial prompt:** Contains ticket content. Agent calls `pickupTicket` to start.
**MCP Tools:** Read ticket (for reference), pickup, submit report, approve

## Ticket Lifecycle

```
[backlog] ──spawn──► [progress] ──approve──► [done]
                          │
                          └── session ends without approval
                              (ticket stays in progress, can re-spawn)
```

## Ticket JSON Schema

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "title": "Add login functionality",
  "body": "## Problem\n\nUsers cannot...\n\n## Requirements\n\n- ...",

  "dates": {
    "created": "2026-01-18T10:00:00Z",
    "updated": "2026-01-18T14:00:00Z",
    "approved": null
  },

  "sessions": [
    {
      "id": "sess-uuid-1",
      "started_at": "2026-01-18T11:00:00Z",
      "ended_at": "2026-01-18T13:00:00Z",
      "agent": "claude",
      "tmux_window": "add-login-func",
      "git_base": {
        ".": "abc123def"
      },
      "report": {
        "files": ["src/components/Login.tsx"],
        "scope_changes": null,
        "decisions": ["Started with JWT approach"],
        "summary": "Partial implementation"
      },
      "status_history": [
        {"status": "in_progress", "tool": "Edit", "work": "Writing login form", "at": "2026-01-18T12:00:00Z"},
        {"status": "starting", "tool": null, "work": null, "at": "2026-01-18T11:00:00Z"}
      ]
    },
    {
      "id": "sess-uuid-2",
      "started_at": "2026-01-18T14:00:00Z",
      "ended_at": null,
      "agent": "claude",
      "tmux_window": "add-login-func",
      "git_base": {
        ".": "def456abc"
      },
      "report": {
        "files": [
          "src/components/Login.tsx",
          "src/hooks/useAuth.ts"
        ],
        "scope_changes": "Added password reset flow (not in original ticket)",
        "decisions": [
          "Used JWT instead of session cookies for stateless auth",
          "Added rate limiting to prevent brute force"
        ],
        "summary": "Implemented login with JWT auth and rate limiting"
      },
      "current_status": {
        "status": "in_progress",
        "tool": "Edit",
        "work": "Adding error handling",
        "at": "2026-01-18T15:30:00Z"
      },
      "status_history": [
        {"status": "in_progress", "tool": "Read", "work": "Reading auth module", "at": "2026-01-18T14:30:00Z"},
        {"status": "starting", "tool": null, "work": null, "at": "2026-01-18T14:00:00Z"}
      ]
    }
  ]
}
```

**Note:** Session state is derived from ticket status (which folder) and `dates.approved`. No explicit outcome field needed.

### Tmux Naming Convention

- `tmux_session`: Project name (e.g., `cortex1`)
- `tmux_window`: Slugified title, max 20 chars (e.g., `add-login-func`)

Window 0 is reserved for architect session.

### Agent Status Values

- `starting` - Session just spawned
- `in_progress` - Agent actively working
- `idle` - Agent waiting for input
- `waiting_permission` - Agent waiting for tool permission
- `error` - Agent encountered error

## MCP Tools

### Architect Session Tools

| Tool | Parameters | Description |
|------|------------|-------------|
| `listTickets` | `status?: string` | List tickets, optionally by status |
| `searchTickets` | `query: string, filters?: object` | Search by title, keyword, date |
| `readTicket` | `id: string` | Read full ticket content |
| `createTicket` | `title: string, body: string` | Create ticket in backlog |
| `updateTicket` | `id: string, title?: string, body?: string` | Update ticket |
| `deleteTicket` | `id: string` | Delete ticket |
| `moveTicket` | `id: string, to: status` | Move ticket between statuses |
| `spawnSession` | `id: string` | Start tmux + agent for ticket |
| `getSessionStatus` | `id: string` | Get agent status for ticket |

### Ticket Session Tools

| Tool | Parameters | Description |
|------|------------|-------------|
| `readTicket` | (none, uses env) | Read own ticket (for long conversations) |
| `pickupTicket` | (none) | Signal starting work, runs `on_pickup` hooks |
| `submitReport` | `report: object` | Update report, runs `on_submit` hooks |
| `approve` | `commit_message: string` | Approve and conclude, runs `on_approve` hooks |

**Report object:**
```json
{
  "files": ["src/login.ts", "src/auth.ts"],
  "scope_changes": "Added feature X not in original ticket",
  "decisions": ["Used approach A because..."],
  "summary": "What was accomplished"
}
```

### Hook Response Format

All MCP tools that run hooks return:

```json
{
  "success": true,
  "hooks": [
    {"command": "npm run lint", "stdout": "✓ All files pass", "exit_code": 0},
    {"command": "npm run test", "stdout": "42 tests passed", "exit_code": 0}
  ]
}
```

On failure:
```json
{
  "success": false,
  "hooks": [
    {"command": "./scripts/validate.sh", "stdout": "Please use conventional commit format", "exit_code": 1}
  ]
}
```

## Lifecycle Hooks

Defined in `.cortex/cortex.yaml`. Hooks are shell commands that run sequentially.

```yaml
lifecycle:
  on_pickup:
    # Inform agent about branch convention
    - run: "echo 'Please create a feature branch named feat/{{ticket_slug}}'"
    # Or automate it
    # - run: "git checkout -b feat/{{ticket_slug}}"

  on_submit:
    - run: "npm run lint"
    - run: "npm run test"

  on_approve:
    # Validate commit message format
    - run: "./scripts/validate-commit.sh '{{commit_message}}'"
    # Commit and push
    - run: "git add -A"
    - run: "git commit -m '{{commit_message}}'"
    - run: "git push -u origin HEAD"
```

### Hook Behavior

- Commands run sequentially
- stdout is captured and returned to agent
- exit 0 = success, continue to next hook
- exit 1+ = failure, stop and return output to agent
- Agent sees both output and exit code, can react accordingly

### Template Variables

| Variable | Available In | Description |
|----------|--------------|-------------|
| `{{ticket_id}}` | All hooks | Ticket UUID |
| `{{ticket_slug}}` | All hooks | Slugified title (e.g., `add-login-func`) |
| `{{ticket_title}}` | All hooks | Full ticket title |
| `{{commit_message}}` | `on_approve` | Message passed to `approve()` |

## Project Configuration

```yaml
# .cortex/cortex.yaml

name: my-project
agent: claude                    # claude | opencode

git:
  repos:
    - path: "."                  # Main repo
    - path: "packages/shared"    # Monorepo package
    # Daemon discovers .git in each path

lifecycle:
  on_pickup:
    - run: "echo 'Create branch feat/{{ticket_slug}} before starting'"
  on_submit:
    - run: "make lint"
    - run: "make test"
  on_approve:
    - run: "./scripts/validate-commit.sh '{{commit_message}}'"
    - run: "git add -A && git commit -m '{{commit_message}}' && git push"
```

## TUI Views

### Kanban View (`cortex kanban`)

```
┌─────────────────────────────────────────────────────────────────┐
│ cortex1                                              [a]rchitect│
├───────────────────┬───────────────────┬─────────────────────────┤
│ Backlog (3)       │ Progress (1)      │ Done (5)                │
├───────────────────┼───────────────────┼─────────────────────────┤
│ > Add auth        │ • Fix bug         │   Cleanup               │
│   API docs        │   (working...)    │   Setup CI              │
│   Logging         │                   │   Refactor auth         │
│                   │                   │   ...                   │
├───────────────────┴───────────────────┴─────────────────────────┤
│ [s]pawn  [Enter] View  [h/l] Move column  [j/k] Navigate       │
└─────────────────────────────────────────────────────────────────┘
```

### Session View (`cortex session <id>`) - Read Only

```
┌─────────────────────────────────────────────────────────────────┐
│ Ticket: Add login functionality                      [progress] │
├─────────────────────────────────────────────────────────────────┤
│ ## Problem                                                      │
│ Users cannot log in to the application...                       │
│                                                                 │
│ ## Requirements                                                 │
│ - Email/password login                                          │
│ - JWT tokens                                                    │
├─────────────────────────────────────────────────────────────────┤
│ Report                                                          │
│ ├── Files (4)                                                   │
│ │   src/components/Login.tsx                                   │
│ │   src/hooks/useAuth.ts                                       │
│ │   ...                                                        │
│ ├── Decisions                                                   │
│ │   • Used JWT for stateless auth                              │
│ │   • Added rate limiting                                      │
│ └── Summary                                                     │
│     Implemented login with JWT auth and rate limiting          │
├─────────────────────────────────────────────────────────────────┤
│ Git Repos                                    [j/k] nav  [o] open│
│ > .                feat/add-login   +142 -38  4 files           │
│   packages/shared  feat/add-login   +12 -5   2 files            │
├─────────────────────────────────────────────────────────────────┤
│ Session: sess-uuid-2  Status: in_progress  [f]ocus tmux  [q]uit│
└─────────────────────────────────────────────────────────────────┘
```

**Navigation:**
- `j/k` - Navigate repos list
- `o` - Open diff tool (configurable: `git diff` or `lazygit` via `settings.yaml`)
- `f` - Focus tmux window
- `q` - Quit

**Note:** Session view is read-only. All actions (approve, request changes) happen through the agent pane. User chats with agent directly.

## Agent Integration

### Spawning Claude with MCP

```bash
# Architect session
claude \
  --mcp-config /tmp/cortex-architect-mcp.json \
  "You are the architect for $PROJECT_NAME. Use your tools to manage tickets."

# Ticket session
claude \
  --mcp-config /tmp/cortex-ticket-mcp.json \
  --prompt "$(cat <<EOF
# Ticket: {{ticket_title}}

{{ticket_body}}

---
Call pickupTicket to start, then implement the requirements.
When done, call submitReport with your changes.
When user approves, call approve with a conventional commit message.
EOF
)"
```

**Note:** No tools are disallowed. Lifecycle hooks can instruct the agent to commit/push as needed (e.g., `echo "Please commit and push your changes"`), or hooks can automate it.

### MCP Config File

```json
{
  "mcpServers": {
    "cortex": {
      "command": "cortexd",
      "args": ["mcp", "--ticket-id", "{{ticket_id}}"],
      "env": {
        "CORTEX_PROJECT_PATH": "{{project_path}}"
      }
    }
  }
}
```

## Daemon API

### Tickets

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/tickets/{status}` | List tickets by status (backlog, progress, done) |
| POST | `/tickets` | Create ticket (always in backlog) |
| GET | `/tickets/{status}/{id}` | Get ticket |
| PUT | `/tickets/{status}/{id}` | Update ticket |
| DELETE | `/tickets/{status}/{id}` | Delete ticket |
| POST | `/tickets/{status}/{id}/move` | Move to different status |
| POST | `/tickets/{status}/{id}/spawn` | Spawn session (moves to progress) |

### Sessions

| Method | Endpoint | Description |
|--------|----------|-------------|
| DELETE | `/sessions/{id}` | Kill session (tmux window) |

**Note:** Active sessions are derived from `/tickets/progress`. Each ticket in progress with an active session has session data in its JSON.

### MCP (called by agent tools)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/mcp/pickup` | Run on_pickup hooks |
| POST | `/mcp/submit` | Update report, run on_submit hooks |
| POST | `/mcp/approve` | Run on_approve hooks, move to done |

### Other

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |

## File Structure

```
cortex1/
├── cmd/
│   ├── cortex/              # CLI + TUI
│   │   ├── main.go
│   │   ├── tui/
│   │   └── commands/
│   └── cortexd/             # Daemon
│       ├── main.go
│       ├── api/             # HTTP handlers
│       ├── mcp/             # MCP server
│       └── core/            # Business logic
├── internal/
│   ├── ticket/              # Ticket JSON operations
│   ├── lifecycle/           # Hook execution
│   ├── git/                 # Git repo discovery & diff
│   └── tmux/                # Tmux session management
├── pkg/
│   └── version/
├── LINEAGE.md
├── DESIGN.md
├── REMOTE.md
├── CLAUDE.md
├── go.mod
└── Makefile
```

## Migration Path

1. Use `cortex0` to bootstrap development
2. Implement daemon core (ticket CRUD, lifecycle)
3. Implement MCP server
4. Implement TUI
5. Self-host: use `cortex1` to develop `cortex1`
