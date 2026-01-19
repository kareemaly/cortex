# Cortex Lineage

## Ancestor

**Location:** `~/projects/cortex` (now `cortex0`)
**Repository:** github.com/kareemaly/cortex (archived)

## What Cortex v0 Was

An orchestration layer for AI coding workflows:

```
┌─────────────────┐     HTTP :4200     ┌─────────────────┐
│   cortex CLI    │ ◄────────────────► │  cortex-daemon  │
│   (Go/TUI)      │                    │  (Go/SQLite)    │
└─────────────────┘                    └─────────────────┘
        │                                      │
        ▼                                      ▼
   tmux sessions                         ~/.cortex/
   (agent + yazi)                        ├── cortex.db
                                         ├── settings.yaml
                                         └── templates/
```

### Session Types
- **Ticket sessions**: Tied to markdown files in `tickets/`
- **Workbench sessions**: Ad-hoc tasks with descriptions
- **Architect sessions**: Planning and backlog management

### Key Components
- SQLite database for all state
- Git wrapper intercepting commits/pushes
- Heavy prompt templates instructing agents
- Claude Code hooks for status updates
- TUI with kanban board and session views

## What We Learned

### Worked Well
- TUI-based workflow (kanban, session views)
- Tmux integration for agent sessions
- Lifecycle concept (spawn → work → review → approve)
- Daemon/CLI separation

### Pain Points
- **SQLite complexity**: Session state, events, git actions all in DB
- **Git wrapper fragility**: Intercepting git commands was error-prone
- **Heavy prompting**: Agents often ignored or misinterpreted instructions
- **Three session types**: Workbench was rarely used, added complexity
- **Orphan state**: DB could get out of sync with reality

## What We're Keeping

| Component | Status |
|-----------|--------|
| Go daemon + CLI | ✓ Keep |
| TUI (bubbletea) | ✓ Keep |
| Tmux integration | ✓ Keep |
| Ticket lifecycle | ✓ Keep (simplified) |
| Claude Code hooks | ✓ Keep (modified) |
| Architect sessions | ✓ Keep (singleton) |

## What We're Dropping

| Component | Reason |
|-----------|--------|
| SQLite database | Tickets become JSON files |
| Git wrapper | Daemon discovers repos directly |
| Workbench sessions | Unnecessary complexity |
| Heavy prompt templates | MCP tools define behavior |
| Events table | Git history + daemon.log |
| Machines table | Git handles multi-machine |
| Complex session states | Simple: backlog → progress → review → done |

## Evolution

```
cortex v0 (2024-2025)
├── Learned: TUI workflow, tmux integration, lifecycle gates
├── Problem: Too much state, fragile git wrapper, prompt-dependent
│
└── cortex v1 (2026)
    ├── File-based tickets (JSON)
    ├── MCP tools instead of prompts
    ├── No git wrapper
    └── Simpler lifecycle
```
