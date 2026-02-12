---
id: b809e23e-a18c-414f-b5d9-b064baf175e7
author: claude
type: done
created: 2026-02-11T10:09:44.21833Z
---
## Summary

### What was done

1. **Upgraded OpenCode** from v1.1.18 to v1.1.56 (latest) via `npm install -g --prefix` to work around a nvm/pyenv npm prefix conflict.

2. **Validated Cortex agent spawn with GPT 5.2** — tested three approaches, all successful:
   - `opencode run --agent cortex` (non-interactive, headless)
   - `opencode --agent cortex --prompt "..."` (interactive TUI with auto-submitted first message)
   - `OPENCODE_CONFIG_CONTENT` env var for fully inline config (no files needed)

3. **Validated MCP server integration** — Cortex MCP server (`cortexd mcp`) works with OpenCode both via `opencode.json` file and inline `OPENCODE_CONFIG_CONTENT`.

### Key findings

- **Model ID**: `openai/gpt-5.2` (also available: `gpt-5.2-codex`, `gpt-5.2-pro`, `gpt-5.2-chat-latest`)
- **Agent config**: `mode: "primary"` is required for `--agent` flag usage; subagents are ignored with fallback
- **`OPENCODE_CONFIG_CONTENT`**: Supports full inline JSON for agent + MCP config, but does NOT support `{file:path}` or `{env:VAR}` substitution — prompts must be fully inlined
- **MCP config format**: Uses `"command": ["cortexd", "mcp"]` (array, not string) and `"environment"` (not `"env"`) with `"type": "local"` (not `"stdio"`)
- **Interactive mode**: Use `opencode --agent X --prompt "..."` (not `opencode run`) for interactive TUI sessions
- **Environment info** (cwd, platform, date) is always appended to the system prompt — cannot be suppressed without a plugin hook

### No code changes
Tooling upgrade + integration validation only. No source files modified.