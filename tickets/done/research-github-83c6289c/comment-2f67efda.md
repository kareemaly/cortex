---
id: 2f67efda-e452-49ee-a0e0-adb89d35619d
author: claude
type: done
created: 2026-02-05T10:03:08.595595Z
---
## Research Complete: GitHub Copilot CLI Integration

### Key Findings

1. **Two Copilot CLIs Exist**
   - `gh copilot` (GitHub CLI extension) - Simple shell helper, not suitable
   - `copilot` (Standalone) - Full agentic CLI similar to Claude Code, this is the target

2. **High Integration Feasibility**
   - MCP config format is compatible (both use `mcpServers` JSON structure)
   - Session resume functionality works similarly (`--resume <id>`)
   - Non-interactive mode available (`-p "prompt" --yolo`)
   - Multi-model support: Claude Sonnet/Haiku/Opus, GPT-5.x, Gemini

3. **Key Difference: Prompt Delivery**
   - Claude: `--system-prompt` and `--append-system-prompt` CLI flags
   - Copilot: Uses AGENTS.md files, no CLI flag for system prompts
   - **Decision**: For Copilot agents, skip SYSTEM.md entirely and put all workflow guidance in KICKOFF.md

### Agreed Architecture

**Defaults Structure** (separate folders per agent):
```
~/.cortex/defaults/
├── claude-code/    # Existing - unchanged
│   └── prompts/    # SYSTEM.md + KICKOFF.md
└── copilot/        # NEW
    └── prompts/    # KICKOFF.md only (includes MCP workflow)
```

**Copilot KICKOFF.md** must include:
- Task context (ticket details)
- MCP tool inventory and descriptions
- Workflow guidance (when to call each tool)

### Required Implementation Changes

1. **`internal/project/config/config.go`**
   - Add `AgentCopilot AgentType = "copilot"`
   - Update `Validate()` to accept copilot

2. **`internal/core/spawn/launcher.go`**
   - Add `buildCopilotLauncherScript()` function
   - Map flags: `--additional-mcp-config`, `--yolo`, `--no-custom-instructions`

3. **`internal/core/spawn/spawn.go`**
   - Skip SYSTEM.md loading when agent=copilot
   - Route to correct launcher based on agent type

4. **`internal/defaults/copilot/`**
   - New embedded defaults with KICKOFF.md templates
   - `cortex.yaml` with `agent: copilot` and `args: [--yolo]`

5. **`internal/install/`**
   - Update `cortex defaults upgrade` to install copilot defaults

### Estimated Scope
~300-400 lines of code across 5-6 files, plus new prompt templates.

### Next Steps
1. Create implementation ticket from this research
2. Start with config + launcher changes (core plumbing)
3. Add copilot defaults with KICKOFF.md templates
4. Test MCP tool compatibility with Copilot CLI