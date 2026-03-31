# Cortex

[![Release](https://img.shields.io/github/v/release/kareemaly/cortex)](https://github.com/kareemaly/cortex/releases/latest)

Orchestration layer for AI coding agents. File-based ticket management with MCP tools and tmux session management.

## What is Cortex?

Cortex turns your AI coding assistant into a managed development team. An architect agent breaks down work into tickets, spawns worker agents in isolated tmux sessions, reviews their output, and approves changes—all through a kanban-style workflow.

## Quick Start

```bash
# Install (latest stable)
curl -fsSL https://github.com/kareemaly/cortex/releases/latest/download/install.sh | bash

# Initialize a new architect workspace
cortex init myproject

# Start the architect
cd myproject && cortex architect start
```

The architect will guide you through creating and managing tickets.

## Requirements

- **tmux** - session management
- **git** - version control
- **AI agent** (one of):
  - [Claude CLI](https://claude.ai/code) - recommended
  - [OpenCode](https://github.com/opencode-ai/opencode)

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
| `cortex init <name>` | Initialize architect workspace |
| `cortex architect start [name]` | Start or attach to architect session |
| `cortex architect list` | List registered architects |
| `cortex architect show [name]` | Project TUI |
| `cortex daemon status` | Check daemon status |
| `cortex upgrade` | Refresh embedded defaults |
| `cortex eject <path>` | Customize a default prompt |

## Configuration

### Global Config

`~/.cortex/settings.yaml` - daemon settings and default agent variants (auto-populated on first `cortex init`):

```yaml
port: 4200
bind_address: 127.0.0.1  # use 0.0.0.0 for remote VM deployments
log_level: info

agents:
  claude-opus:
    agent: claude
    args: ["--dangerously-skip-permissions"]
  claude-opus-plan:
    agent: claude
    args: ["--allow-dangerously-skip-permissions", "--permission-mode", "plan"]
  claude-sonnet:
    agent: claude
    args: ["--dangerously-skip-permissions", "--model", "claude-sonnet-4-6"]
  claude-sonnet-plan:
    agent: claude
    args: ["--allow-dangerously-skip-permissions", "--permission-mode", "plan", "--model", "claude-sonnet-4-6"]
  opencode:
    agent: opencode
    args: []
  opencode-plan:
    agent: opencode
    args: ["--agent", "plan"]
```

Global agents are inherited by every architect project. To add or override variants for a specific project, define them in `cortex.yaml` — project-level entries take precedence over global ones.

### Project Config

`cortex.yaml` - project-specific settings:

```yaml
name: my-project
repos:
  - ~/projects/my-repo

# Optional: add project-specific agent variants or override global ones
# agents:
#   my-variant:
#     agent: claude
#     args: ["--dangerously-skip-permissions", "--model", "claude-opus-4-5"]
```

After running `cortex init`, edit `cortex.yaml` to:
- Add your repos under `repos:`
- Set `companion: lazygit` for side-by-side git UI
- Optionally add project-specific agent variants under `agents:`

### Customizing Prompts

Use `cortex eject` to copy default prompts for customization:

```bash
cortex eject work/KICKOFF.md        # Customize ticket workflow
cortex eject architect/SYSTEM.md    # Customize architect behavior
```

Ejected prompts go to `<project>/prompts/`. Un-ejected prompts fall back to `~/.cortex/defaults/main/prompts/`.

## How It Works

1. **Initialize** - `cortex init` creates the workspace with config and ticket storage
2. **Daemon starts** - `cortexd` launches automatically, serves all projects on port 4200
3. **Architect session** - AI agent with MCP tools for ticket management
4. **Worker sessions** - Each ticket gets a dedicated tmux window with scoped MCP tools
5. **Review cycle** - Workers request review, architect approves

The daemon handles:
- Ticket storage (YAML frontmatter + markdown in `tickets/`)
- Session records (persistent conclusion records in `sessions/`)
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
