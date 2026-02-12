---
id: 35b28fba-5cbd-4602-9ef2-65c6d54e01e1
author: claude
type: done
created: 2026-02-11T08:27:11.235834Z
---
Completed comprehensive research on OpenCode CLI v1.1.18 (npm package `opencode-ai`). Created a detailed research doc covering all 9 questions. Key findings:

1. **Config format**: `opencode.json` or `opencode.jsonc` (JSON/JSONC). Config merges across global (~/.config/opencode/opencode.json), project (./opencode.json), env vars (OPENCODE_CONFIG, OPENCODE_CONFIG_CONTENT), and .opencode/ directories.

2. **MCP servers**: Configured under `mcp` key. Local servers use `type: "local"` with `command` (array) and `environment` (object). Remote servers use `type: "remote"` with `url`. OAuth is auto-handled.

3. **Custom instructions**: Auto-reads AGENTS.md, CLAUDE.md, CONTEXT.md via findUp pattern. Additional files via `instructions` config array. Agent-level prompts via markdown files in .opencode/agents/.

4. **--prompt flag**: Pre-fills and auto-submits a prompt into the TUI on startup. TUI still renders for continued interaction.

5. **--agent flag**: Selects which primary agent to use. Falls back to default "build" agent if not found or if subagent specified.

6. **opencode run**: Non-interactive mode. Starts headless server, sends prompt, streams output, exits when idle. Does NOT auto-approve permissions - uses interactive CLI prompts for permission decisions.

7. **No --yolo flag**: Auto-approve is achieved via permission config: `"permission": {"*": "allow"}`. The default "build" agent already has most permissions allowed, except doom_loop, external_directory, and .env reads.

8. **Context files auto-read**: AGENTS.md (primary), CLAUDE.md, CONTEXT.md locally; ~/.config/opencode/AGENTS.md and ~/.claude/CLAUDE.md globally.

9. **Programmatic system prompts**: Via agent markdown files (.opencode/agents/*.md), JSON config with `{file:...}` references, OPENCODE_CONFIG_CONTENT env var, or `instructions` config field.

Files changed: None (research-only task). Doc created in cortex docs store.