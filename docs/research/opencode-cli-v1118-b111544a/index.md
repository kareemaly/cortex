---
id: b111544a-08d4-4153-a2f9-dddea1a025ea
title: OpenCode CLI v1.1.18 (npm) Configuration Reference
tags:
    - opencode
    - cli
    - configuration
    - mcp
    - agents
created: 2026-02-11T08:25:30.461924Z
updated: 2026-02-11T08:25:30.461924Z
---
## Overview

OpenCode (npm package `opencode-ai`, currently at v1.1.18 locally, v1.1.56 latest) is a TypeScript-based AI coding agent built for the terminal. It was rewritten from the original Go-based tool (now archived as "Crush" by Charm). The current version is maintained at `github.com/anomalyco/opencode` by thdxr (Dax Raad / SST team). It compiles to a native binary via Bun.

---

## 1. Config File Format

**Format**: JSON or JSONC (JSON with Comments)

**File names**: `opencode.json` or `opencode.jsonc`

**Config precedence** (lowest to highest):
1. Remote config (`.well-known/opencode`)
2. Global config: `~/.config/opencode/opencode.json`
3. Custom config via `OPENCODE_CONFIG` env var
4. Project config: `opencode.json` in project root
5. `.opencode/` directories (agents, commands, tools, plugins)
6. Inline config via `OPENCODE_CONFIG_CONTENT` env var

**Config files merge** rather than replace each other.

**Variable substitution** is supported:
- `{env:VARIABLE_NAME}` - Environment variables
- `{file:path/to/file}` - File contents (relative or absolute)

---

## 2. MCP Server Configuration

MCP servers are configured under the `mcp` key in `opencode.json`:

```jsonc
{
  "mcp": {
    // Local (stdio) MCP server
    "my-local-mcp": {
      "type": "local",
      "command": ["npx", "-y", "@modelcontextprotocol/server-everything"],
      "environment": {
        "MY_VAR": "value"
      },
      "enabled": true,     // optional, default true
      "timeout": 5000      // optional, tool fetch timeout in ms
    },

    // Remote (HTTP/SSE) MCP server
    "my-remote-mcp": {
      "type": "remote",
      "url": "https://my-mcp-server.com",
      "headers": {
        "Authorization": "Bearer {env:MY_API_KEY}"
      },
      "oauth": {
        "clientId": "{env:CLIENT_ID}",
        "clientSecret": "{env:CLIENT_SECRET}",
        "scope": "tools:read tools:execute"
      },
      "enabled": true,
      "timeout": 5000
    }
  }
}
```

**Key differences from Claude Code MCP config**:
- Uses `command` as an **array** (not a string), e.g. `["node", "server.js"]`
- Uses `environment` (not `env`) for environment variables
- Uses `type: "local"` instead of `type: "stdio"`
- Remote servers use `type: "remote"` with a `url` field
- OAuth is handled automatically by OpenCode (Dynamic Client Registration RFC 7591)

**Disabling MCP tools globally or per-agent**:
```jsonc
{
  "tools": {
    "server-name": false,       // disable all tools from a server
    "mymcp_*": false            // glob pattern to disable tools
  }
}
```

**CLI management**: `opencode mcp add`, `opencode mcp list`, `opencode mcp auth <name>`, `opencode mcp logout <name>`, `opencode mcp debug <name>`

---

## 3. Custom Instructions / System Prompts

### Context Files (Auto-Read)

OpenCode automatically reads instruction files using a `findUp` pattern from the working directory to the project root:

**Local rule files** (searched via `findUp`, first match wins per file name):
1. `AGENTS.md` (primary, preferred)
2. `CLAUDE.md` (compatibility with Claude Code projects)
3. `CONTEXT.md`

**Global rule files**:
1. `~/.config/opencode/AGENTS.md`
2. `~/.claude/CLAUDE.md` (compatibility)

If `OPENCODE_CONFIG_DIR` is set, it also looks for `AGENTS.md` in that directory.

The system prompt is assembled from:
- `SystemPrompt.header()` - provider-specific header
- `SystemPrompt.environment()` - environment context (OS, shell, project info)
- `SystemPrompt.custom()` - custom instructions from the files above
- `SystemPrompt.instructions()` - additional instruction files from config
- `SystemPrompt.provider()` - provider-specific prompt

### Additional Instructions via Config

```jsonc
{
  "instructions": ["./my-rules.md", "**/.cursorrules", ".github/copilot-instructions.md"]
}
```

The `instructions` field accepts an array of file paths or glob patterns. These are loaded and included in the system prompt.

### Agent-Level System Prompts

Each agent can have its own system prompt defined as a markdown file. See the Agents section below.

### Programmatic System Prompt Configuration

To set a system prompt programmatically:
1. Create an agent markdown file in `.opencode/agents/` with the desired prompt as the body
2. Use `OPENCODE_CONFIG_CONTENT` env var to inject config with agent definitions
3. Use the `instructions` config field to point to additional instruction files
4. Use `{file:path/to/prompt.txt}` syntax in config values

---

## 4. The `--prompt` Flag (TUI Default Command)

```
opencode [project] --prompt "some prompt text"
```

When used with the default TUI command, `--prompt` pre-fills and auto-submits a prompt into the TUI on startup. The TUI opens normally, but the prompt is automatically sent as the first message. If stdin is piped (not a TTY), the piped content is prepended to the `--prompt` value.

This is **NOT** the same as `opencode run` (non-interactive mode). The TUI still renders and the user can continue interacting after the initial prompt completes.

---

## 5. The `--agent` Flag

Available on both the default TUI command and `opencode run`:

```
opencode --agent build
opencode run --agent plan "Analyze this codebase"
```

Specifies which **primary** agent to use for the session. If the specified agent is not found or is a subagent, it falls back to the default agent (typically "build") with a warning.

Built-in agents:
- **build** (primary) - Default, full-access agent with `"*": "allow"` permissions
- **plan** (primary) - Read-only agent, denies edits by default, asks before bash
- **compaction** (primary) - Used for context compaction
- **explore** (subagent) - Read-only codebase exploration
- **general** (subagent) - Full tool access for multi-step tasks

Use `opencode agent list` to see all available agents with their permissions.

---

## 6. How `opencode run` Works

```
opencode run [message..] [--options]
```

### Behavior:
- Accepts a message as positional arguments (or via stdin pipe)
- Starts a headless server internally (or attaches to existing one via `--attach`)
- Sends the prompt to the agent
- Streams output to stdout (formatted text by default, or raw JSON with `--format json`)
- **Exits when the session goes idle** (agent finishes processing)
- Exits with code 1 if there's an error

### Permission Handling:
- `opencode run` does **NOT** auto-approve all permissions
- It uses interactive CLI prompts (via `@clack/core`) to ask the user for permission decisions
- Each permission request shows three options: "Allow once", "Always allow", "Reject"
- When creating a session, it sets `question: deny` (disables the Question tool that would ask the user questions)

### Auto-Approve Equivalent:
There is **no `--yolo` flag**. To achieve auto-approval, configure permissions in the agent config or the `opencode.json`:

```jsonc
{
  "permission": {
    "*": "allow"
  }
}
```

Or at the agent level:
```jsonc
{
  "agent": {
    "my-agent": {
      "permission": {
        "*": "allow"
      }
    }
  }
}
```

The default **build** agent already has `"*": "allow"` as its base permission rule, so in practice most tool calls are auto-approved. The exceptions that still prompt:
- `doom_loop` (infinite tool call loops) - set to `"ask"`
- `external_directory` (files outside project) - set to `"ask"`
- Reading `.env` files - set to `"ask"`

### Key Flags:
| Flag | Purpose |
|------|---------|
| `--model` / `-m` | Model in `provider/model` format |
| `--agent` | Which agent to use |
| `--command` | Run a custom command, use message as args |
| `--file` / `-f` | Attach file(s) to message |
| `--format` | `default` (formatted) or `json` (raw JSON events) |
| `--continue` / `-c` | Continue last session |
| `--session` / `-s` | Continue specific session by ID |
| `--attach` | Attach to running `opencode serve` instance |
| `--title` | Session title |
| `--variant` | Model variant (reasoning effort: high, max, minimal) |
| `--port` | Port for local server |

---

## 7. Auto-Approve / Yolo Mode

**There is no `--yolo` flag.**

However, the permission system is highly configurable. The default "build" agent already has most permissions set to `"allow"`. To make it fully auto-approve everything:

```jsonc
// opencode.json
{
  "permission": {
    "*": "allow",
    "doom_loop": "allow",
    "external_directory": "allow"
  }
}
```

Or per-agent in markdown format (`.opencode/agents/myagent.md`):
```markdown
---
permission:
  "*": allow
  doom_loop: allow
  external_directory: allow
---
Your system prompt here.
```

Permission types: `bash`, `edit`, `read`, `write`, `glob`, `grep`, `list`, `webfetch`, `websearch`, `codesearch`, `task`, `external_directory`, `doom_loop`, `question`

Permission actions: `"allow"`, `"deny"`, `"ask"`

Bash permissions support glob patterns for granular control:
```jsonc
{
  "permission": {
    "bash": {
      "*": "ask",
      "git status *": "allow",
      "grep *": "allow",
      "rm -rf *": "deny"
    }
  }
}
```

---

## 8. Context Files Auto-Read

**Local rule files** (found via `findUp` from cwd to project root):
- `AGENTS.md` (primary)
- `CLAUDE.md` (Claude Code compatibility)
- `CONTEXT.md`

**Global rule files**:
- `~/.config/opencode/AGENTS.md`
- `~/.claude/CLAUDE.md`

**Additional instruction sources** (via config):
- `instructions` array in `opencode.json` - glob patterns for extra instruction files
- `.github/copilot-instructions.md` and `.cursorrules` are recognized during `/init` (AGENTS.md generation) but are not auto-loaded unless listed in `instructions`

**AGENTS.md scoping rules** (from the system prompt):
- The scope of an AGENTS.md file is the entire directory tree rooted at the folder containing it
- More-deeply-nested AGENTS.md files take precedence in case of conflicts
- Direct system/developer/user instructions take precedence over AGENTS.md

---

## 9. Programmatic System Prompt Configuration

Multiple approaches:

### A. Agent markdown file
Create `.opencode/agents/myagent.md`:
```markdown
---
description: My custom agent
mode: primary
model: anthropic/claude-sonnet-4-20250514
permission:
  "*": allow
---
Your full system prompt goes here as the markdown body.
It supports all markdown formatting.
```

### B. JSON config with prompt file reference
```jsonc
{
  "agent": {
    "myagent": {
      "description": "My custom agent",
      "mode": "primary",
      "prompt": "{file:./prompts/myagent-prompt.txt}"
    }
  }
}
```

### C. OPENCODE_CONFIG_CONTENT env var
```bash
OPENCODE_CONFIG_CONTENT='{"agent":{"myagent":{"description":"Custom","prompt":"You are..."}}}' opencode run --agent myagent "do something"
```

### D. Instructions config field
```jsonc
{
  "instructions": ["./my-custom-rules.md", "./prompts/*.md"]
}
```

### E. `opencode agent create` CLI
Interactive wizard that generates an agent markdown file with a system prompt:
```bash
opencode agent create --description "Reviews code" --mode primary --tools "bash,read,glob,grep"
```

---

## Summary of Key Paths

| Item | Path |
|------|------|
| Global config | `~/.config/opencode/opencode.json` |
| Global data | `~/.local/share/opencode/` |
| Global cache | `~/.cache/opencode/` |
| Global agents | `~/.config/opencode/agents/*.md` |
| Global instructions | `~/.config/opencode/AGENTS.md` |
| Project config | `./opencode.json` or `./opencode.jsonc` |
| Project agents | `./.opencode/agents/*.md` |
| Project commands | `./.opencode/commands/*.md` |
| Project tools | `./.opencode/tool/*.ts` |
| Project plugins | `./.opencode/plugin/*.ts` |
| Project themes | `./.opencode/themes/*.json` |

## Key Environment Variables

| Variable | Purpose |
|----------|---------|
| `OPENCODE_CONFIG` | Path to custom config file |
| `OPENCODE_CONFIG_CONTENT` | Inline JSON config content |
| `OPENCODE_CONFIG_DIR` | Custom config directory |
| `OPENCODE_CLIENT` | Client type: `cli` (default), `tui` |
| `OPENCODE_AUTO_SHARE` | Auto-share sessions |
| `OPENCODE_DISABLE_CLAUDE_CODE` | Disable Claude Code prompt/skills |
| `OPENCODE_DISABLE_CLAUDE_CODE_PROMPT` | Disable Claude Code prompt only |
| `OPENCODE_DISABLE_CLAUDE_CODE_SKILLS` | Disable Claude Code skills only |
| `OPENCODE_ENABLE_EXPERIMENTAL_MODELS` | Enable alpha models |
| `OPENCODE_FAKE_VCS` | Override VCS detection |
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `OPENAI_API_KEY` | OpenAI API key |
| `GEMINI_API_KEY` | Google Gemini API key |