---
id: 0e53c209-bff0-4b02-8eca-dfcb259b836e
author: claude
type: comment
created: 2026-02-05T10:02:34.479414Z
---
## Correction: Defaults Architecture

The defaults are agent-specific folders. We create a NEW `copilot` defaults folder, leaving `claude-code` untouched.

### Directory Structure
```
~/.cortex/defaults/
├── claude-code/              # EXISTING - unchanged
│   ├── cortex.yaml           # agent: claude
│   └── prompts/
│       ├── architect/
│       │   ├── SYSTEM.md     ✓
│       │   └── KICKOFF.md    ✓
│       └── ticket/work/
│           ├── SYSTEM.md     ✓
│           └── KICKOFF.md    ✓
│
└── copilot/                  # NEW
    ├── cortex.yaml           # agent: copilot, args: [--yolo]
    └── prompts/
        ├── architect/
        │   └── KICKOFF.md    # Full workflow + MCP docs
        └── ticket/work/
            └── KICKOFF.md    # Full workflow + MCP docs
```

### Project Config
```yaml
# .cortex/cortex.yaml
extend: ~/.cortex/defaults/copilot
# OR
extend: ~/.cortex/defaults/claude-code
```

### Implementation
1. Create `internal/defaults/copilot/` embedded files
2. Add to `cortex defaults upgrade` command
3. Code changes to skip SYSTEM.md loading when agent=copilot
4. Launcher changes to build copilot command