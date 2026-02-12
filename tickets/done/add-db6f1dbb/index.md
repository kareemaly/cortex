---
id: db6f1dbb-fe79-4a38-a5c0-a9fd6ab55785
title: Add buildOpenCodeCommand to spawn launcher
type: work
tags:
    - opencode
created: 2026-02-11T10:28:55.77967Z
updated: 2026-02-11T10:47:27.799148Z
---
## Objective

Add OpenCode-specific command building to the spawn launcher so that `opencode` agent type uses the correct CLI invocation instead of being treated as claude.

## Context

Currently in `internal/core/spawn/launcher.go`, the `opencode` agent type falls through to the same `buildClaudeCommand()` as claude. This is wrong — OpenCode has a completely different CLI interface.

The validated working command pattern is:
```bash
OPENCODE_CONFIG_CONTENT='{"agent":{"cortex":{"description":"Cortex ticket agent","mode":"primary","model":"openai/gpt-5.2","prompt":"<SYSTEM.md content>","permission":{"*":"allow"}}},"mcp":{"cortex":{"type":"local","command":["cortexd","mcp"],"environment":{"CORTEX_PROJECT_PATH":"...","CORTEX_DAEMON_URL":"..."}}}}' \
  opencode --agent cortex --prompt "<KICKOFF content>"
```

## What to build

### `buildOpenCodeCommand()` in launcher.go

A new function that:

1. **Packs the SYSTEM prompt** into the agent's `prompt` field inside `OPENCODE_CONFIG_CONTENT` JSON
2. **Embeds MCP config** into the `mcp` key of the config content (the MCP server should point to `cortexd mcp` with appropriate env vars — look at how the existing MCP config JSON is generated for claude)
3. **Sets permissions** — `"permission": {"*": "allow"}` for full auto-approve
4. **Sets the agent definition** — `mode: "primary"`, description, and the system prompt as the `prompt` field
5. **Builds the command** — `opencode --agent cortex --prompt "$(cat <kickoff_file>)"`
6. **Exports `OPENCODE_CONFIG_CONTENT`** as an environment variable in the launcher script

### Update routing

Update the agent type routing in `GenerateLauncherScript()` (or wherever the agent type switch happens) so `opencode` calls `buildOpenCodeCommand()` instead of `buildClaudeCommand()`.

### Key differences from Claude
- No `--system-prompt` or `--append-system-prompt` flags — system prompt goes into `OPENCODE_CONFIG_CONTENT`
- No `--mcp-config` flag — MCP config goes into `OPENCODE_CONFIG_CONTENT`
- No `--settings` flag — not applicable
- No `--resume` support — OpenCode always starts fresh
- No `--session-id` — not applicable
- Uses `--agent cortex` to select the dynamically defined agent
- Uses `--prompt "<kickoff>"` to auto-submit the first message in TUI mode

### MCP config format for OpenCode
```json
{
  "mcp": {
    "cortex": {
      "type": "local",
      "command": ["cortexd", "mcp"],
      "environment": {
        "CORTEX_PROJECT_PATH": "<project_path>",
        "CORTEX_DAEMON_URL": "http://localhost:4200",
        "CORTEX_TICKET_ID": "<ticket_id>",
        "CORTEX_SESSION_ID": "<session_id>"
      }
    }
  }
}
```

Note the format differences from Claude's MCP config: `command` is an array (not string), `environment` (not `env`), `type: "local"` (not `stdio`).

## Acceptance criteria
- `buildOpenCodeCommand()` exists and generates the correct launcher script
- `opencode` agent type routes to the new builder
- Generated `OPENCODE_CONFIG_CONTENT` includes system prompt, MCP config, and permissions
- Launcher script correctly exports the env var and runs `opencode --agent cortex --prompt "..."`
- Existing claude and copilot launch paths are unaffected