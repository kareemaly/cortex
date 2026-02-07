---
id: d8d305ca-07ad-42e8-9ee4-950e0c637af2
author: claude
type: comment
created: 2026-02-05T09:48:25.234311Z
---
## Initial Discovery: Two Copilot CLIs

Found two distinct Copilot CLI tools:

### 1. `gh copilot` (GitHub CLI Extension)
- Simple command-line helper
- Commands: `suggest`, `explain`, `config`, `alias`
- Purpose: Generate shell commands, explain existing commands
- Limited for agent workflows

### 2. `copilot` (Standalone Agentic CLI)
- Full-featured AI coding assistant (similar to Claude Code!)
- Supports multiple models: Claude (Sonnet/Haiku/Opus), GPT-5.x, Gemini
- Has MCP server support built-in
- Uses `AGENTS.md` for custom instructions (like Claude's `CLAUDE.md`)
- Session management with resume/continue
- Tool permissions system
- Non-interactive mode with `--prompt` flag
- Agent Client Protocol support via `--acp`

The standalone `copilot` CLI is the interesting one for Cortex integration.