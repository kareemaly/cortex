# Cortex

[![Release](https://img.shields.io/github/v/release/kareemaly/cortex)](https://github.com/kareemaly/cortex/releases/latest)

Orchestration layer for AI coding agents. File-based ticket management with MCP tools and tmux session management.

## What is Cortex?

Cortex turns your AI coding assistant into a managed development team. An architect agent breaks down work into tickets, spawns worker agents in isolated tmux sessions, reviews their output, and approves changes—all through a kanban-style workflow.

## Quick Start

```bash
# Install (latest stable)
curl -fsSL https://github.com/kareemaly/cortex/releases/latest/download/install.sh | bash

# Initialize in your project
cd your-project && cortex init

# Start the architect
cortex architect
```

The architect will guide you through creating and managing tickets.

## Requirements

- **tmux** - session management
- **git** - version control
- **Claude CLI** - [claude.ai/code](https://claude.ai/code)

## Core Workflow

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│   Architect                                                     │
│   ├── Creates tickets in backlog                                │
│   ├── Spawns worker agents for tickets                          │
│   ├── Reviews completed work                                    │
│   └── Approves → ticket moves to done                           │
│                                                                 │
│   Worker Agent (per ticket)                                     │
│   ├── Runs in isolated tmux window                              │
│   ├── Implements the ticket                                     │
│   ├── Calls requestReview when done                             │
│   └── Waits for architect approval                              │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Ticket lifecycle**: `backlog → progress → review → done`

## Commands

| Command | Description |
|---------|-------------|
| `cortex init` | Initialize project with `.cortex/` directory |
| `cortex architect` | Start or attach to architect session |
| `cortex kanban` | Kanban board TUI |
| `cortex show [id]` | Ticket detail view |
| `cortex ticket list` | List all tickets |
| `cortex ticket spawn <id>` | Manually spawn agent for ticket |
| `cortex projects` | List all registered projects |
| `cortex daemon status` | Check daemon status |
| `cortex upgrade` | Self-update to latest version |
| `cortex eject <path>` | Customize a default prompt |

## Configuration

### Project Config

`.cortex/cortex.yaml` - project-specific settings:

```yaml
name: my-project
extend: ~/.cortex/defaults/claude-code
architect:
  agent: claude
  args: ["--allowedTools", "mcp__cortex__*"]
ticket:
  work:     # Default implementation workflow
    agent: claude
    args: ["--permission-mode", "plan"]
  debug:    # Root cause analysis workflow
    agent: claude
  research: # Read-only exploration workflow
    agent: claude
  chore:    # Quick maintenance workflow
    agent: claude
git:
  worktrees: false  # Enable git worktrees for ticket isolation
```

### Global Config

`~/.cortex/settings.yaml` - daemon and notification settings:

```yaml
daemon:
  port: 4200
  log_level: info
notifications:
  enabled: true
```

### Customizing Prompts

Use `cortex eject` to copy default prompts for customization:

```bash
cortex eject ticket/work/SYSTEM.md    # Customize ticket workflow
cortex eject architect/SYSTEM.md      # Customize architect behavior
```

Ejected prompts go to `.cortex/prompts/`.

See [CONFIG_DOCS.md](internal/install/defaults/claude-code/CONFIG_DOCS.md) for full configuration reference.

## How It Works

1. **Initialize** - `cortex init` creates `.cortex/` with config and ticket storage
2. **Daemon starts** - `cortexd` launches automatically, serves all projects on port 4200
3. **Architect session** - AI agent with MCP tools for ticket management
4. **Worker sessions** - Each ticket gets a dedicated tmux window with scoped MCP tools
5. **Review cycle** - Workers request review, architect approves

The daemon handles:
- Ticket storage (JSON files in `.cortex/tickets/`)
- Tmux session orchestration
- MCP server for AI agents
- SSE events for real-time updates

## Development

```bash
make build    # Build binaries
make lint     # Run linter
make test     # Run tests
make install  # Build and install to ~/.local/bin/
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup.

## Architecture

Single `cortexd` daemon serves all projects over HTTP. All clients (CLI, TUI, MCP) communicate through the daemon API—no direct file access. This enables running the daemon on a remote VM with local clients.

```
┌─────────────────┐
│  cortex CLI/TUI │──┐
└─────────────────┘  │
┌─────────────────┐  │  HTTP :4200   ┌──────────────────────┐
│  MCP Architect  │──┼──────────────▶│      cortexd         │
└─────────────────┘  │               │  ├─ HTTP API         │
┌─────────────────┐  │               │  ├─ Ticket Store     │
│  MCP Ticket     │──┘               │  ├─ Tmux management  │
└─────────────────┘                  │  └─ SSE events       │
                                     └──────────────────────┘
```

See [CLAUDE.md](CLAUDE.md) for detailed architecture and code paths.

## License

MIT
