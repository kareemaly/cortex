---
id: 593e9fac-5b1a-4c74-905d-4f9db93ea269
author: claude
type: comment
created: 2026-02-05T09:49:30.085691Z
---
## Copilot CLI Architecture Analysis

### Custom Instructions (AGENTS.md equivalent)
- Uses multiple file locations for instructions that **combine** (not override):
  - Personal: `~/.copilot/` directory
  - Repository: `.github/copilot-instructions.md` or similar
  - Organization: enterprise/org level config
- Supports **skills** in `.agents/skills/` directory (auto-loaded)
- Single-file commands in `.claude/commands/` (simpler alternative)

### MCP Server Support
- Built-in GitHub MCP server enabled by default
- Configuration via `~/.copilot/mcp-config.json`
- Per-session config with `--additional-mcp-config <json>`
- Supports OAuth 2.0 authentication for remote servers
- `/mcp show` command to inspect configured servers

### Non-Interactive Mode (Key for Cortex Integration!)
```bash
copilot -p "your prompt" --allow-all-tools
copilot -p "your prompt" --yolo  # equivalent shorthand
```

### Available Models
- Claude: sonnet-4.5, haiku-4.5, opus-4.5, sonnet-4
- GPT: 5.2-codex, 5.2, 5.1-codex-max, 5.1-codex, 5.1, 5, 5-mini
- Gemini: 3-pro-preview

### Recent Notable Features (from changelog)
- Agent Client Protocol (ACP) server mode via `--acp`
- Plugin marketplace and custom agents
- Session resume/continue functionality
- Autopilot mode (experimental) for autonomous task completion
- Skills system for reusable prompts/workflows