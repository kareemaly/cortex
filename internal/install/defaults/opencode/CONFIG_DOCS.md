# OpenCode Configuration

Configuration guide for Cortex projects using opencode agent type.

## Project Config Schema

Project configuration lives in `.cortex/cortex.yaml`:

```yaml
name: my-project                      # Project display name
extend: ~/.cortex/defaults/opencode   # Inherit from base config
architect:
  agent: opencode
ticket:
  work:
    agent: opencode
git:
  worktrees: false  # Enable git worktrees for ticket isolation
```

| Field | Description |
|-------|-------------|
| `name` | Project display name |
| `extend` | Path to base config for inheritance |
| `architect.agent` | Agent type (`opencode`) |
| `architect.args` | CLI arguments passed to architect agent |
| `ticket.<type>.agent` | Agent type for ticket type |
| `ticket.<type>.args` | CLI arguments for ticket agent |
| `git.worktrees` | Enable git worktrees per ticket |

## Prompt Structure

| Path | Purpose |
|------|---------|
| `architect/SYSTEM.md` | Role definition and workflow rules |
| `architect/KICKOFF.md` | Initial context with ticket list |
| `ticket/work/SYSTEM.md` | Ticket agent workflow instructions |
| `ticket/work/KICKOFF.md` | Ticket details and context |
| `ticket/work/APPROVE.md` | Post-approval commit instructions |
| `ticket/work/REJECT.md` | Rollback instructions |

## Customizing Prompts

Use `cortex eject` to copy a default prompt for project-level customization:

```bash
cortex eject <prompt-path>

# Examples
cortex eject ticket/work/SYSTEM.md    # Customize ticket workflow
cortex eject architect/SYSTEM.md      # Customize architect behavior
cortex eject ticket/work/APPROVE.md   # Customize approval flow
```

Ejected prompts are placed in `.cortex/prompts/`:

```
.cortex/prompts/
├── architect/
│   ├── SYSTEM.md      # Override architect system prompt
│   └── KICKOFF.md     # Override architect kickoff
└── ticket/
    └── work/
        ├── SYSTEM.md  # Override ticket system prompt
        ├── KICKOFF.md # Override ticket kickoff
        └── APPROVE.md # Override approval instructions
```

**Resolution order**: Project prompts → Extended base.

## Template Variables

### Architect Prompts

| Variable | Description |
|----------|-------------|
| `{{.ProjectName}}` | Project name from config |
| `{{.TicketList}}` | Formatted ticket list by status |

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

### Add Test Requirements

Override `.cortex/prompts/ticket/work/APPROVE.md`:

```markdown
## Approved
1. Run tests: `make test`
2. Commit and push
3. Call `concludeSession` with a summary
```

### Custom Kickoff Context

Override `.cortex/prompts/ticket/work/KICKOFF.md`:

```markdown
# Ticket: {{.TicketTitle}}
{{.TicketBody}}

## Project Guidelines
- Follow existing code patterns
- Write tests for new functionality
```

## Commands Reference

| Command | Description |
|---------|-------------|
| `cortex init --agent opencode` | Initialize project with opencode defaults |
| `cortex eject <path>` | Copy prompt to project for customization |
| `cortex architect` | Start or attach to architect session |
| `cortex kanban` | Open kanban board TUI |
