---
id: 3e6b150d-27b2-49a4-b525-ea1be8ceb63d
author: claude
type: comment
created: 2026-02-11T08:16:38.023215Z
---
Completed comprehensive research on OpenCode CLI and its successor Crush. Created doc `c8081eab` with findings covering all 9 research areas. Key findings:

1. **Config**: JSON format (`.opencode.json` / `.crush.json`), MCP servers configured in config file
2. **No --system-prompt flag**: Uses context path files (opencode.md, AGENTS.md, CLAUDE.md) auto-injected into prompts
3. **Headless mode**: `-p` flag (OpenCode) or `run` subcommand (Crush), exits on completion
4. **No session resume**: Each invocation creates a new session, no CLI session management
5. **No --agent flag**: Internal agent types only (coder, task, summarizer, title)
6. **Crush adds**: `--yolo` flag, `--model` override, Agent Skills (`SKILL.md`), HTTP MCP transport, `system_prompt_prefix` per provider
7. **Integration strategy**: Generate config file + context file before spawn, launch in interactive/TUI mode in tmux pane, use MCP tools for lifecycle