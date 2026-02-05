# Copilot CLI Configuration

Configuration guide for Cortex projects using GitHub Copilot CLI agent type.

## Project Config Schema

Project configuration lives in `.cortex/cortex.yaml`:

```yaml
name: my-project                     # Project display name
extend: ~/.cortex/defaults/copilot   # Inherit from base config
architect:
  agent: copilot
  args: ["--yolo"]
ticket:
  work:
    agent: copilot
    args: ["--yolo"]
git:
  worktrees: false  # Enable git worktrees for ticket isolation
```

| Field | Description |
|-------|-------------|
| `name` | Project display name |
| `extend` | Path to base config for inheritance |
| `architect.agent` | Agent type (`copilot`) |
| `architect.args` | CLI arguments passed to architect agent |
| `ticket.<type>.agent` | Agent type for ticket type |
| `ticket.<type>.args` | CLI arguments for ticket agent |
| `git.worktrees` | Enable git worktrees per ticket |

## Copilot CLI Flags

| Flag | Description |
|------|-------------|
| `--yolo` | Required for automation mode |
| `--no-custom-instructions` | Ignore AGENTS.md files (set by Cortex) |
| `--additional-mcp-config` | MCP server configuration (set by Cortex) |
| `--model` | Model selection (e.g., `--model gpt-4o`) |
| `--resume` | Resume a previous session |

## Key Differences from Claude

1. **No System Prompt Support**: Copilot doesn't support `--system-prompt`. All workflow guidance is included in KICKOFF.md files instead of SYSTEM.md.

2. **No Settings/Hooks**: Copilot doesn't support `--settings` for lifecycle hooks. Hook functionality is not available.

3. **MCP Config Flag**: Uses `--additional-mcp-config` instead of `--mcp-config`.

4. **Model Selection**: Configure via args (e.g., `args: ["--yolo", "--model", "gpt-4o"]`).

## Prompt Structure

| Path | Purpose |
|------|---------|
| `architect/KICKOFF.md` | Full architect workflow + MCP tool docs |
| `ticket/work/KICKOFF.md` | Ticket details + workflow guidance |
| `ticket/work/APPROVE.md` | Post-approval commit instructions |

**Note**: There are no SYSTEM.md files for Copilot. All instructions go in KICKOFF.md.

## Customizing Prompts

Use `cortex eject` to copy a default prompt for project-level customization:

```bash
cortex eject <prompt-path>

# Examples
cortex eject ticket/work/KICKOFF.md   # Customize ticket workflow
cortex eject architect/KICKOFF.md     # Customize architect behavior
cortex eject ticket/work/APPROVE.md   # Customize approval flow
```

Ejected prompts are placed in `.cortex/prompts/`:

```
.cortex/prompts/
├── architect/
│   └── KICKOFF.md     # Override architect kickoff + workflow
└── ticket/
    └── work/
        ├── KICKOFF.md # Override ticket kickoff + workflow
        └── APPROVE.md # Override approval instructions
```

**Resolution order**: Project prompts → Extended base.

## Template Variables

### Architect Prompts

| Variable | Description |
|----------|-------------|
| `{{.ProjectName}}` | Project name from config |
| `{{.TicketList}}` | Formatted ticket list by status |
| `{{.CurrentDate}}` | Current date, time, and timezone |

### Ticket Prompts

| Variable | Description |
|----------|-------------|
| `{{.ProjectPath}}` | Root project directory |
| `{{.TicketID}}` | Ticket identifier |
| `{{.TicketTitle}}` | Ticket title |
| `{{.TicketBody}}` | Ticket body/description |
| `{{.IsWorktree}}` | Boolean, true when running in worktree |
| `{{.WorktreePath}}` | Path to worktree (when enabled) |
| `{{.WorktreeBranch}}` | Git branch name (when in worktree) |

## Common Customizations

### Enable Git Worktrees

```yaml
git:
  worktrees: true
```

### Select Different Model

```yaml
architect:
  args: ["--yolo", "--model", "gpt-4o"]
ticket:
  work:
    args: ["--yolo", "--model", "gpt-4o"]
```

## Commands Reference

| Command | Description |
|---------|-------------|
| `cortex init` | Initialize project with `.cortex/` directory |
| `cortex eject <path>` | Copy prompt to project for customization |
| `cortex architect` | Start or attach to architect session |
| `cortex kanban` | Open kanban board TUI |

## Limitations

- **No lifecycle hooks**: Copilot doesn't support the `--settings` flag used for hooks
- **No session ID tracking**: Session management is more limited
- **MCP tool format**: Same MCP configuration format works with Copilot
