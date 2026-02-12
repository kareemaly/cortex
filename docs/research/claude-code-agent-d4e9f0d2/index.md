---
id: d4e9f0d2-2474-489b-8be1-110f089859fc
title: Claude Code Agent Integration Pattern
tags:
    - agent-type
    - claude-code
    - integration
    - spawn-system
    - mcp
    - prompt-system
created: 2026-02-11T08:09:57.200348Z
updated: 2026-02-11T08:09:57.200348Z
---
# Claude Code Agent Integration Pattern

Complete documentation of how the `claude-code` agent type is configured, launched, and managed throughout its lifecycle in Cortex.

## Overview

The Claude Code agent system is a three-tier hierarchy:
- **Meta** (global, cross-project)
- **Architect** (project-scoped)
- **Ticket Agent** (ticket-scoped)

Each tier operates independently but shares the same spawn orchestration, configuration, and prompt resolution system.

---

## 1. Configuration System

### Project Configuration (`internal/project/config/config.go`)

Projects define agent behavior in `.cortex/cortex.yaml`:

```yaml
extend: ~/.cortex/defaults/claude-code  # Inherit from base config
name: my-project                         # Used as tmux session name

architect:
  agent: claude                          # Agent type: claude, opencode, copilot
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
  debug:
    agent: claude
    args: [...]
  research:
    agent: claude
    args: [...]
  chore:
    agent: claude
    args: [...]

meta:
  agent: claude
  args: [...]

git:
  worktrees: false                       # Enable git worktrees per ticket

docs:
  path: docs                             # Custom docs directory

tickets:
  path: tickets                          # Custom tickets directory
```

**Key Config Features:**
- **Inheritance**: Projects extend from `~/.cortex/defaults/claude-code` which provides sensible defaults
- **Agent Types**: Each role can use different agents (claude, opencode, copilot)
- **Tool Allowlisting**: CLI args control which MCP tools are available to each agent
- **Role-based Args**: Architect, Meta, and each ticket type can have custom CLI arguments

### Config Loading (`config.Load()` flow)

```
1. Load project config from .cortex/cortex.yaml
2. If "extend" field is set:
   - Resolve extend path (can be ~/cortex/defaults/claude-code)
   - Recursively load base config
   - Merge: base + project overrides
3. Validate merged config
4. Store resolved extend path for prompt fallback
```

The resolved extend path is critical: it's passed through the spawn pipeline and used for prompt fallback resolution (see section 2).

---

## 2. Prompt System

### Prompt Architecture

Three distinct prompt types, each with a role hierarchy:

#### **Ticket Agent Prompts** (`prompts/ticket/{type}/{stage}.md`)
- **Types**: work, debug, research, chore
- **Stages**:
  - `SYSTEM.md` — Role definition, MCP tool instructions, workflow rules
  - `KICKOFF.md` — Dynamic ticket details (template with variables)
  - `APPROVE.md` — Post-approval instructions (run tests, commit, etc.)

#### **Architect Prompts** (`prompts/architect/{stage}.md`)
- **Stages**:
  - `SYSTEM.md` — Role definition, orchestration rules, ticket quality standards
  - `KICKOFF.md` — Dynamic ticket list and project context (template with variables)

#### **Meta Prompts** (`prompts/meta/{stage}.md`)
- **Stages**:
  - `SYSTEM.md` — Global admin role definition, cross-project orchestration
  - `KICKOFF.md` — Dynamic project list and active sessions (template with variables)

### Resolution Order

**For Ticket/Architect Prompts:**
```
1. Check project's .cortex/prompts/{role}/{type}/{stage}.md (highest priority)
2. Check base config's prompts/{role}/{type}/{stage}.md (fallback)
3. Error if not found
```

**For Meta Prompts:**
```
1. Check base config's prompts/meta/{stage}.md only
   (Meta is global, not per-project)
2. Error if not found
```

Implementation: `internal/prompt/resolver.go`

### Template Variables

**Ticket Agent (`TicketVars`):**
- `{{.ProjectPath}}` — Project root directory
- `{{.TicketID}}` — Full ticket ID
- `{{.TicketTitle}}` — Ticket title
- `{{.TicketBody}}` — Ticket description
- `{{.Comments}}` — Pre-formatted comments block
- `{{.IsWorktree}}` — True if running in git worktree
- `{{.WorktreePath}}` — Git worktree path (when enabled)
- `{{.WorktreeBranch}}` — Feature branch name (when enabled)

**Architect (`ArchitectKickoffVars`):**
- `{{.ProjectName}}` — Project name from config
- `{{.TicketList}}` — Formatted tickets by status (Backlog, Progress, Review, Done)
- `{{.CurrentDate}}` — ISO timestamp
- `{{.TopTags}}` — Comma-separated top 20 tags
- `{{.DocsList}}` — Formatted recent docs (top 20 by creation date)

**Meta (`MetaKickoffVars`):**
- `{{.CurrentDate}}` — ISO timestamp
- `{{.ProjectList}}` — Formatted project list with status and ticket counts
- `{{.SessionList}}` — Active sessions per project

### Rendering

`internal/prompt/template.go` uses Go's `text/template` package to render variables:
```go
func RenderTemplate(content string, vars any) (string, error)
```

This happens during spawn, so dynamic data (tickets, projects, sessions) is current at spawn time.

---

## 3. Spawn Orchestration (`internal/core/spawn/orchestrate.go`)

### State Detection

Before spawning a ticket agent, the system detects three possible states:

```
StateNormal    — No existing session
StateActive    — Tmux window exists and is running
StateOrphaned  — Session record exists but tmux window is gone
```

### Mode Matrix

Based on state and requested mode:

```
| Mode    | Normal        | Active         | Orphaned    |
|---------|---------------|----------------|-------------|
| normal  | Spawn new     | AlreadyActive  | StateError  |
| resume  | StateError    | StateError     | Resume      |
| fresh   | StateError    | StateError     | Fresh       |
```

- **normal**: Default mode. Spawn only if no existing session.
- **resume**: Continue an orphaned session (re-attach to tmux).
- **fresh**: Destroy any existing session and spawn new.

### Orchestrate Function

Entry point: `spawn.Orchestrate(ctx, req, deps)`

**Flow:**
```
1. Validate mode (normal/resume/fresh)
2. Load project config
3. Look up ticket by ID
4. Determine ticket type → lookup agent config
5. Resolve agent type (request > config > "claude")
6. Resolve tmux session name (request > config name)
7. Look up existing session in store
8. Detect current state (Normal/Active/Orphaned)
9. Apply state/mode matrix:
   - Normal + normal → Spawn()
   - Orphaned + resume → Resume()
   - Orphaned + fresh → Fresh()
   - Other combinations → Error
10. On success: Move ticket to Progress (if in Backlog)
11. Return OrchestrateResult
```

Returns:
- `Outcome`: spawned, resumed, or already_active
- `Ticket`: Current ticket state
- `SpawnResult`: Window info, files created
- `StateInfo`: Current session state

---

## 4. Spawn Process (`internal/core/spawn/spawn.go`)

### Spawn Request

```go
type SpawnRequest struct {
  AgentType      AgentType       // architect, ticket_agent, meta
  Agent          string          // "claude", "opencode", "copilot"
  TmuxSession    string
  ProjectPath    string
  TicketsDir     string
  
  // Ticket agent only
  TicketID       string
  Ticket         *ticket.Ticket
  UseWorktree    bool
  
  // Architect agent only
  ProjectName    string
  
  // Shared
  AgentArgs      []string        // CLI args from config
  BaseConfigPath string          // Resolved extend path for prompt fallback
}
```

### Spawn Steps

1. **Validate Request**
   - Check required fields
   - Validate tmux session name (alphanumeric + - _)
   - Check project path exists

2. **Find Cortexd Binary**
   - Lookup via `internal/binpath.FindCortexd()` if not provided
   - Used for MCP server command and hook commands

3. **Generate Tmux Window Name**
   - For ticket agents: slug of ticket title
   - For architect: "architect"
   - For meta: "meta"

4. **Create Worktree (if enabled)**
   - Generate unique session ID
   - Create git worktree via `internal/worktree.Manager`
   - Return worktree path and feature branch name
   - Store in session record

5. **Create Session Record**
   - Call `SessionStore.Create()` to track session
   - Store: TicketID, Agent, TmuxWindow, WorktreePath, FeatureBranch
   - For architect/meta: Use `CreateArchitect()` / `CreateMeta()`

6. **Generate MCP Config** (section 5)

7. **Generate Settings Config** (section 6)

8. **Build Prompts**
   - Load system prompt (if not Copilot)
   - Load and render kickoff template with variables
   - Handle fallbacks if templates don't exist

9. **Write Temp Files**
   - MCP config JSON
   - Settings config JSON
   - Prompt text file
   - System prompt text file
   - Launcher script

10. **Spawn in Tmux**
    - Ticket agents: 30% agent pane, 70% companion (`cortex show`)
    - Architect: Agent pane + companion (`cortex kanban`)
    - Meta: Agent pane + companion (`cortex dashboard`)

11. **Cleanup Trap**
    - Launcher script has trap to clean up all temp files on exit

### Return Value

```go
type SpawnResult struct {
  Success       bool
  TicketID      string
  TmuxWindow    string
  WindowIndex   int
  MCPConfigPath string
  SettingsPath  string
  Message       string
}
```

---

## 5. MCP Configuration (`internal/core/spawn/config.go`)

### Generated Config

JSON file written to temp directory before spawn:

```json
{
  "mcpServers": {
    "cortex": {
      "command": "/path/to/cortexd",
      "args": ["mcp", "--ticket-id", "ABC123", "--ticket-type", "work"],
      "env": {
        "CORTEX_TICKETS_DIR": "/project/tickets",
        "CORTEX_PROJECT_PATH": "/project",
        "CORTEX_TMUX_SESSION": "my-project",
        "CORTEX_DAEMON_URL": "http://127.0.0.1:4200"
      }
    }
  }
}
```

### Cortexd MCP Startup

When Claude starts with `--mcp-config cortex-mcp-*.json`:

1. Claude reads config and spawns `cortexd mcp` subprocess
2. Cortexd loads MCP tools based on args:
   - `--ticket-id ABC123` → Load ticket agent tools (addComment, addBlocker, readReference, requestReview, concludeSession)
   - `--meta` → Load meta session tools
   - No ticket-id/meta args + project path → Load architect tools

3. `CORTEX_PROJECT_PATH` env var: Used by MCP tools to route mutations through HTTP API
4. `CORTEX_TICKET_ID` env var: Set by launcher script, visible to tools

### MCP Tool Scope

**Ticket Agent Tools** (`tools_ticket.go`):
- `readReference` — Read referenced ticket/doc
- `addComment` — Log progress
- `addBlocker` — Report blocker
- `requestReview` — Submit for approval
- `concludeSession` — Mark done, trigger cleanup
- `createDoc` (research only) — Document findings

**Architect Tools** (`tools_architect.go`):
- Ticket CRUD: listTickets, readTicket, createTicket, updateTicket, deleteTicket, moveTicket
- Comments: addTicketComment
- Session: spawnSession (for ticket agents)
- Docs: createDoc, readDoc, updateDoc, deleteDoc, moveDoc, listDocs, addDocComment
- Config: readProjectConfig, updateProjectConfig

**Meta Tools** (`tools_meta.go`):
- Project: listProjects, registerProject, unregisterProject
- Architect: spawnArchitect
- Config: readGlobalConfig, updateGlobalConfig, readProjectConfig, updateProjectConfig
- Prompts: readPrompt, updatePrompt
- Session: listSessions, concludeSession
- Debug: readDaemonLogs, daemonStatus

Tool allowlisting is set via `args: ["--allowedTools", "tool1,tool2,...]` in config.

---

## 6. Settings Configuration (`internal/core/spawn/settings.go`)

Claude-specific hooks configuration (JSON):

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "*",
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/cortexd hook post-tool-use"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/cortexd hook stop"
          }
        ]
      }
    ],
    "PermissionRequest": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/cortexd hook permission-request"
          }
        ]
      }
    ]
  }
}
```

**Hook Points:**
- `PostToolUse`: After each tool call (for logging, state sync)
- `Stop`: When session ends (for cleanup, summary generation)
- `PermissionRequest`: When agent requests permission (for approval handling)

**Why Settings Config?**
- Cortex doesn't support per-agent configuration inheritance
- Settings.json is Claude's hook/preferences mechanism
- Each agent session gets a unique settings file with cortexd hooks

**Important**: Copilot doesn't support `--settings` flag, so settings config is skipped for copilot agents.

---

## 7. Launcher Script (`internal/core/spawn/launcher.go`)

### Script Generation

Bash script generated and written to temp file before spawn.

**For Claude agents:**
```bash
#!/usr/bin/env bash

# Cleanup trap
trap 'rm -f /tmp/cortex-mcp-*.json /tmp/cortex-prompt-*.txt /tmp/cortex-settings-*.json ...' EXIT

# Export environment variables
export CORTEX_TICKET_ID=ABC123
export CORTEX_PROJECT=/project

# Build and execute claude command
claude "$(cat /tmp/cortex-prompt-ticket.txt)" \
  --system-prompt "$(cat /tmp/cortex-sysprompt-ticket.txt)" \
  --mcp-config /tmp/cortex-mcp-ticket.json \
  --settings /tmp/cortex-settings-ticket.json \
  --permission-mode plan \
  --allow-dangerously-skip-permissions \
  --allowedTools mcp__cortex__readReference
```

**For Copilot agents:**
```bash
gh copilot agent \
  --yolo \
  --no-custom-instructions \
  "$(cat /tmp/cortex-prompt-ticket.txt)" \
  --additional-mcp-config /tmp/cortex-mcp-ticket.json \
  --resume session-id
```

**Key Features:**
- `$(cat file)` for prompts — avoids embedding long text in tmux send-keys
- `trap` cleans up all temp files on exit
- Environment variables injected
- Agent args from config appended
- Resume flag support for orphaned sessions
- Agent-specific flag differences (Claude vs Copilot)

---

## 8. Tmux Integration (`internal/core/spawn/spawn.go` lines 925-942)

### Window Layout

**Ticket Agent:**
```
┌─────────────────────────────────────────┐
│ Agent Pane (30%)   │ Companion (70%)     │
│                    │                     │
│ claude ...         │ cortex show         │
│ (running)          │ (ticket details)    │
└─────────────────────────────────────────┘
```

**Architect Agent:**
```
┌─────────────────────────────────────────┐
│ Agent Pane (30%)   │ Companion (70%)     │
│                    │                     │
│ claude ...         │ cortex kanban       │
│ (running)          │ (ticket board)      │
└─────────────────────────────────────────┘
```

**Meta Agent:**
```
┌─────────────────────────────────────────┐
│ Agent Pane (30%)   │ Companion (70%)     │
│                    │                     │
│ claude ...         │ cortex dashboard    │
│ (running)          │ (project overview)  │
└─────────────────────────────────────────┘
```

### Tmux Manager Interface

```go
SpawnAgent(session, windowName, agentCmd, companionCmd, workDir, companionWorkDir) (windowIndex, error)
SpawnArchitect(session, windowName, agentCmd, companionCmd, workDir, companionWorkDir) error
```

- Creates new window in tmux session
- Splits 30/70
- Runs agent command in left pane
- Runs companion command in right pane
- Returns window index for ordering

---

## 9. Environment Variables

### Agent-Visible Variables

Set in launcher script via `export`:

```bash
export CORTEX_TICKET_ID=abc123       # Ticket ID for agent
export CORTEX_PROJECT=/path/to/proj   # Project root (except meta)
```

Used by MCP tools to determine context.

### MCP Server Environment

Set in MCP config JSON for cortexd subprocess:

```json
"env": {
  "CORTEX_TICKETS_DIR": "/project/tickets",
  "CORTEX_PROJECT_PATH": "/project",
  "CORTEX_TMUX_SESSION": "my-project",
  "CORTEX_DAEMON_URL": "http://127.0.0.1:4200"
}
```

These tell cortexd where to find files and how to reach daemon.

---

## 10. Session Lifecycle

### Session Tracking

**Store**: `internal/session/` (ephemeral, in-memory for this session)

```go
type Session struct {
  ID              string     // Unique session ID
  TicketID        string     // Associated ticket
  Agent           string     // Agent type (claude/opencode/copilot)
  TmuxWindow      string     // Window name
  WorktreePath    *string    // Git worktree path (nil if not used)
  FeatureBranch   *string    // Feature branch (nil if not used)
  CreatedAt       time.Time
  Status          string     // Status (active/orphaned/concluded)
}
```

For Architect/Meta:
- Single session per type per project (not per-ticket)
- Stored in `.cortex/sessions.json` (project)
- Meta sessions stored in `~/.cortex/meta-session.json` (global)

### Session State Transitions

```
Spawn/Resume
    ↓
Active (Agent running in tmux)
    ↓
┌───────────────┬──────────────┐
│ (Agent exits) │ (Approved)   │
↓               ↓
Orphaned        Concluded
│               │
│ (Resume)      └─→ Cleanup
└─→ Active          ↓
                    Done
```

### Agent Workflow

**Ticket Agent (from SYSTEM prompt):**
1. Read ticket details (already in KICKOFF)
2. Use `readReference` if needed
3. Implement changes
4. Use `addComment` to log progress
5. Call `requestReview` → Ticket moves to Review status
6. User approves (external, not via MCP)
7. Agent calls `concludeSession` → Trigger cleanup
8. Cleanup: Generate session doc, move ticket to Done

**Architect (from SYSTEM prompt):**
1. Read ticket list (already in KICKOFF)
2. Analyze backlog
3. Create/update tickets
4. Call `spawnSession` for selected ticket → Ticket agent starts
5. Monitor progress
6. Call `concludeSession` when done

**Meta (from SYSTEM prompt):**
1. Read project list (already in KICKOFF)
2. Manage projects (register/unregister)
3. Spawn architects as needed
4. Monitor sessions
5. Call `concludeSession` when done

### Session Cleanup

Triggered by `concludeSession()` MCP tool:
1. Generate session summary doc (research findings, decisions)
2. End session in session store
3. Delete temp files (via trap in launcher script)
4. For ticket agents: Move ticket to Done status
5. For architect/meta: Close tmux window

---

## 11. Integration Points

### HTTP API Handler

`internal/daemon/api/tickets.go` `handleSpawnTicket()`:

```go
// Receive spawn request from cortex CLI/TUI
result, err := spawn.Orchestrate(ctx, spawn.OrchestrateRequest{
  TicketID:    id,
  Mode:        mode,     // "normal" | "resume" | "fresh"
  ProjectPath: projectPath,
}, spawn.OrchestrateDeps{
  Store:       store,
  SessionStore: sessionStore,
  TmuxManager: tmuxManager,
})

// Return result to client
return c.JSON(http.StatusOK, result)
```

### MCP Tool

`internal/daemon/mcp/tools_architect.go` `spawnSession`:

```go
// Agent calls spawnSession MCP tool
result, err := spawn.Orchestrate(ctx, spawn.OrchestrateRequest{
  TicketID:    ticketID,
  Mode:        mode,
  ProjectPath: projectPath,
}, spawn.OrchestrateDeps{
  Store:       store,
  SessionStore: sessionStore,
  TmuxManager: tmuxManager,
})
```

Same orchestration function, different callers.

---

## 12. Default Configuration

### Base Config Location

`internal/install/defaults/claude-code/`

```
cortex.yaml
prompts/
  architect/
    SYSTEM.md
    KICKOFF.md
  ticket/
    work/
      SYSTEM.md
      KICKOFF.md
      APPROVE.md
    debug/
      ...
    research/
      ...
    chore/
      ...
  meta/
    SYSTEM.md
    KICKOFF.md
```

### Default Agent Configuration

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

### Prompt Philosophy

- **SYSTEM**: Static role definition, workflow rules, tool instructions
- **KICKOFF**: Dynamic context (ticket details, project state) + static instructions
- **APPROVE**: Post-approval instructions (testing, committing, wrapping up)

System prompts use `--system-prompt` (full replace) for architect/meta, `--append-system-prompt` (append to Claude's default) for ticket agents.

---

## 13. Key Differences from OpenCode

The same spawn orchestration system works for OpenCode agents:

1. **Agent Type**: `agent: opencode` in config
2. **Launcher Script**: Uses `claude` command with same flags (compatible CLI)
3. **MCP Config**: Identical JSON format
4. **Settings Config**: Skipped (OpenCode may not support)
5. **Prompts**: Same system/kickoff/approve structure

The separation of spawn orchestration from agent type allows adding new agents (Copilot, custom) without changing spawn logic.

---

## 14. Testing

- **Unit tests**: `internal/core/spawn/spawn_test.go`
- **Integration tests**: `internal/daemon/api/integration_test.go`
- **Config tests**: `internal/project/config/config_test.go`
- **Prompt tests**: `internal/prompt/resolver_test.go`, `template_test.go`

---

## 15. Error Handling

### Validation Errors
- Missing required fields (TicketID, TmuxSession, etc.)
- Invalid tmux session names
- Project path doesn't exist

### State Errors
- Trying to spawn when agent already active
- Trying to resume non-existent session
- State/mode matrix violations

### Runtime Errors
- Prompt load failures (NotFoundError)
- Temp file creation failures
- Tmux spawn failures
- Config loading failures

Soft failures return `SpawnResult.Success = false` with descriptive message. Hard failures return error.

---

## Key Files Reference

| Component | Files |
|-----------|-------|
| Spawn system | `internal/core/spawn/spawn.go`, `orchestrate.go`, `launcher.go` |
| Config | `internal/project/config/config.go`, `merge.go` |
| Prompts | `internal/prompt/resolver.go`, `prompt.go`, `template.go` |
| MCP config | `internal/core/spawn/config.go`, `settings.go` |
| Defaults | `internal/install/defaults/claude-code/` |
| HTTP API | `internal/daemon/api/tickets.go`, `sessions.go` |
| MCP tools | `internal/daemon/mcp/tools_architect.go`, `tools_ticket.go`, `tools_meta.go` |
| Tmux | `internal/tmux/` |
| Session store | `internal/session/` |
| Worktree | `internal/worktree/` |

