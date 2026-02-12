---
id: 908244d4-87b5-4920-8a9c-559770035522
title: Create OpenCode defaults (prompts + cortex.yaml)
type: work
tags:
    - opencode
created: 2026-02-11T10:28:36.673347Z
updated: 2026-02-11T10:42:53.522143Z
---
## Objective

Create the default configuration and prompt files for the OpenCode agent type, mirroring the existing structure used by `claude-code` and `copilot` defaults.

## What to build

Create a new directory at `internal/install/defaults/opencode/` containing:

### `cortex.yaml`
- Agent type: `opencode` for all roles (architect, meta, ticket types)
- Args: minimal or empty — OpenCode permissions are handled via `OPENCODE_CONFIG_CONTENT`, not CLI flags
- Same structure as `claude-code/cortex.yaml` and `copilot/cortex.yaml`

### Prompt files
Same directory structure as claude-code:
```
prompts/
├── architect/
│   ├── SYSTEM.md
│   └── KICKOFF.md
├── ticket/
│   ├── work/
│   │   ├── SYSTEM.md
│   │   ├── KICKOFF.md
│   │   └── APPROVE.md
│   ├── debug/
│   │   ├── SYSTEM.md
│   │   ├── KICKOFF.md
│   │   └── APPROVE.md
│   ├── research/
│   │   ├── SYSTEM.md
│   │   ├── KICKOFF.md
│   │   └── APPROVE.md
│   └── chore/
│       ├── SYSTEM.md
│       ├── KICKOFF.md
│       └── APPROVE.md
└── meta/
    ├── SYSTEM.md
    └── KICKOFF.md
```

- **KICKOFF.md and APPROVE.md** templates can be reused as-is from claude-code — they use the same `{{ .TicketTitle }}`, `{{ .TicketBody }}`, `{{ .Comments }}` template variables
- **SYSTEM.md** prompts can also be reused from claude-code — the role instructions and MCP tool docs are agent-agnostic
- Check the existing claude-code and copilot defaults to understand the patterns

### Embed the defaults
Make sure the new defaults directory is included in the Go embed directive so it gets compiled into the binary. Look at how claude-code and copilot defaults are embedded.

## Acceptance criteria
- `internal/install/defaults/opencode/` exists with all prompt files and cortex.yaml
- Defaults are embedded in the binary (embed directive updated)
- Prompt content is appropriate for the OpenCode agent (reuse from claude-code where applicable)