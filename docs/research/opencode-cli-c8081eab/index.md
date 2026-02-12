---
id: c8081eab-ea00-4579-91f5-11cdad2b1738
title: 'OpenCode CLI Research: Integration Guide for Cortex'
tags:
    - opencode
    - research
    - integration
    - agent
created: 2026-02-11T08:16:27.56454Z
updated: 2026-02-11T08:16:27.56454Z
---
## Overview

OpenCode is a Go-based terminal AI assistant (TUI built with Bubble Tea) supporting multiple LLM providers. The original repository at `github.com/opencode-ai/opencode` is now **archived** and the project has been succeeded by **Crush** (`github.com/charmbracelet/crush`), developed by the original author and the Charm team. Both tools share the same fundamental architecture but Crush has evolved significantly.

This document covers both OpenCode (archived) and Crush (active successor) since the integration approach may need to target whichever version is installed.

---

## 1. Configuration File Format

### OpenCode (archived)

- **Format**: JSON (`.opencode.json`)
- **Search order** (first found wins):
  1. `$HOME/.opencode.json` (global)
  2. `$XDG_CONFIG_HOME/opencode/.opencode.json` (XDG global)
  3. `./.opencode.json` (local/project)
- Local config is **merged** into global config
- Data directory: `.opencode/` (relative to project root, configurable via `data.directory`)

```json
{
  "$schema": "./opencode-schema.json",
  "data": { "directory": ".opencode" },
  "providers": {
    "anthropic": { "apiKey": "$ANTHROPIC_API_KEY", "disabled": false }
  },
  "agents": {
    "coder": { "model": "claude-4-sonnet", "maxTokens": 5000 }
  },
  "mcpServers": {
    "cortex": {
      "type": "stdio",
      "command": "/path/to/cortex-mcp",
      "args": ["--ticket", "abc123"],
      "env": ["CORTEX_PROJECT_PATH=/path/to/project"]
    }
  },
  "contextPaths": ["CLAUDE.md", "opencode.md"],
  "shell": { "path": "/bin/bash", "args": ["-l"] },
  "autoCompact": true,
  "debug": false
}
```

### Crush (successor)

- **Format**: JSON (`.crush.json` or `crush.json`)
- **Search order**:
  1. `.crush.json` (project-specific)
  2. `crush.json` (project-level)
  3. `$HOME/.config/crush/crush.json` (global)
- Override via `CRUSH_GLOBAL_CONFIG` and `CRUSH_GLOBAL_DATA` env vars
- Data directory: `.crush/` (configurable via `options.data_directory`)

```json
{
  "$schema": "https://charm.land/crush.json",
  "models": {
    "large": { "model": "claude-4-sonnet", "provider": "anthropic" },
    "small": { "model": "claude-3.5-haiku", "provider": "anthropic" }
  },
  "providers": {
    "anthropic": { "api_key": "$ANTHROPIC_API_KEY" }
  },
  "mcp": {
    "cortex": {
      "type": "stdio",
      "command": "/path/to/cortex-mcp",
      "args": ["--ticket", "abc123"],
      "env": { "CORTEX_PROJECT_PATH": "/path/to/project" },
      "timeout": 30
    }
  },
  "options": {
    "context_paths": ["AGENTS.md", "CLAUDE.md"],
    "data_directory": ".crush",
    "initialize_as": "AGENTS.md"
  },
  "permissions": {
    "allowed_tools": ["view", "ls", "grep", "bash"]
  }
}
```

**Key difference**: Crush uses `snake_case` JSON keys and the `mcp` key (not `mcpServers`). It also separates model selection from provider configuration and uses `env` as a map instead of an array.

---

## 2. MCP Server Configuration

### OpenCode

Configured under `mcpServers` key. Supports two transport types:

| Field | Type | Description |
|-------|------|-------------|
| `command` | string | Executable path for stdio servers |
| `args` | string[] | Arguments to pass to the command |
| `env` | string[] | Environment variables as `KEY=VALUE` strings |
| `type` | string | `"stdio"` or `"sse"` |
| `url` | string | URL for SSE servers |
| `headers` | map | HTTP headers for SSE servers |

Each MCP tool is namespaced with the server name: `{serverName}_{toolName}`.

MCP clients are initialized at startup with a 30-second timeout. Each tool invocation creates a new client connection (connects, initializes, calls tool, closes). This is a notable inefficiency -- every tool call goes through the full MCP handshake.

### Crush

Configured under `mcp` key. Adds HTTP transport and more options:

| Field | Type | Description |
|-------|------|-------------|
| `command` | string | Executable for stdio servers |
| `args` | string[] | Arguments |
| `env` | map[string]string | Environment variables (map, not array) |
| `type` | string | `"stdio"`, `"sse"`, or `"http"` |
| `url` | string | URL for HTTP/SSE servers |
| `headers` | map | HTTP headers |
| `disabled` | bool | Disable this server |
| `disabled_tools` | string[] | Disable specific tools |
| `timeout` | int | Timeout in seconds (default 15) |

Environment variables support shell expansion: `$(echo $VAR)`.

---

## 3. System Prompts and Custom Instructions

### Neither OpenCode nor Crush has a `--system-prompt` CLI flag.

Instead, they use **context paths** -- files that are automatically read and appended to the system prompt.

### OpenCode Default Context Paths

```go
var defaultContextPaths = []string{
    ".github/copilot-instructions.md",
    ".cursorrules",
    ".cursor/rules/",
    "CLAUDE.md",
    "CLAUDE.local.md",
    "opencode.md",
    "opencode.local.md",
    "OpenCode.md",
    "OpenCode.local.md",
    "OPENCODE.md",
    "OPENCODE.local.md",
}
```

### Crush Default Context Paths

```go
var defaultContextPaths = []string{
    ".github/copilot-instructions.md",
    ".cursorrules",
    ".cursor/rules/",
    "CLAUDE.md",
    "CLAUDE.local.md",
    "GEMINI.md",
    "gemini.md",
    "crush.md",
    "crush.local.md",
    "Crush.md",
    "Crush.local.md",
    "CRUSH.md",
    "CRUSH.local.md",
    "AGENTS.md",
    "agents.md",
    "Agents.md",
}
```

Custom context paths can be configured:
- **OpenCode**: `"contextPaths": ["path1", "path2"]` in `.opencode.json`
- **Crush**: `"options": { "context_paths": ["path1", "path2"] }` in `.crush.json`

Files found at these paths are read and injected into the system prompt under a `# Project-Specific Context` section. Paths ending with `/` are treated as directories and walked recursively. Duplicate files (case-insensitive) are deduplicated.

### How to Inject Custom Instructions for Cortex

**Strategy**: Write a file at one of the default context paths (e.g., `opencode.md` or `AGENTS.md`) or configure `contextPaths` in the config. The content of this file will be injected into every agent prompt automatically.

**Alternative**: For Crush, per-provider custom prompts are possible via `system_prompt_prefix` in the provider config:

```json
{
  "providers": {
    "anthropic": {
      "system_prompt_prefix": "You are a Cortex ticket agent. Follow the MCP tools for workflow."
    }
  }
}
```

---

## 4. Non-Interactive / Headless Mode (`run` command)

### OpenCode

Uses the `-p` / `--prompt` flag on the root command:

```bash
opencode -p "Your prompt here"
opencode -p "Your prompt" -f json   # JSON output
opencode -p "Your prompt" -q        # Quiet (no spinner)
```

**Behavior**:
- Creates a new session automatically
- Auto-approves ALL permission requests (no human confirmation needed)
- Runs the coder agent with the given prompt
- Outputs the final response to stdout
- Exits when done
- Supports `text` and `json` output formats

### Crush

Uses a `run` subcommand:

```bash
crush run "Your prompt here"
crush run --quiet "Your prompt"
crush run --verbose "Your prompt"
crush run --model claude-4-sonnet "Your prompt"
crush run --small-model claude-3.5-haiku "Your prompt"
```

**Additional features**:
- Supports stdin piping: `cat file.txt | crush run "Summarize this"`
- Supports `--model` / `-m` flag to override model for this run
- Supports `--small-model` flag for the task/summarizer agent
- Supports `--verbose` / `-v` for log output to stderr

**Both tools exit after the prompt is processed** -- they do not maintain a persistent session in non-interactive mode.

---

## 5. Session Management

### OpenCode

Sessions are stored in a **SQLite database** (not flat files). The database is in the `.opencode/` directory.

- **Create**: `Ctrl+N` in interactive mode, or automatically on `--prompt` runs
- **Switch**: `Ctrl+A` opens session selector dialog
- **List**: Programmatic only via `session.Service.List()`
- **No CLI commands** for session management
- **No session resume** -- sessions cannot be continued from the CLI
- **Auto-compact**: When context window reaches 95% capacity, automatically summarizes and creates a continuation

Sessions store: ID, title, parent session ID, message count, token counts, cost, summary message ID.

### Crush

Same SQLite-based approach, with some improvements:
- Session data stored in `.crush/` data directory
- Same keyboard shortcuts for interactive mode
- `crush projects` command lists registered projects
- No CLI session management commands (create/resume/list)

### Implication for Cortex

**There is no way to programmatically resume a session** in either tool. Each invocation with `-p` / `run` creates a new session. If the agent needs to continue working, it must be given all context in the initial prompt. The auto-compact feature helps with long-running interactive sessions but does not apply to headless mode.

---

## 6. CLI Flags Reference

### OpenCode

| Flag | Short | Description |
|------|-------|-------------|
| `--help` | `-h` | Show help |
| `--version` | `-v` | Show version |
| `--debug` | `-d` | Enable debug logging |
| `--cwd` | `-c` | Set working directory |
| `--prompt` | `-p` | Non-interactive single prompt |
| `--output-format` | `-f` | Output format: `text` or `json` |
| `--quiet` | `-q` | Hide spinner in non-interactive mode |

### Crush

| Flag | Short | Scope | Description |
|------|-------|-------|-------------|
| `--help` | `-h` | root | Show help |
| `--debug` | `-d` | persistent | Enable debug logging |
| `--cwd` | `-c` | persistent | Set working directory |
| `--data-dir` | `-D` | persistent | Custom data directory |
| `--yolo` | `-y` | root | Auto-accept all permissions |
| | | | |
| `--quiet` | `-q` | `run` | Hide spinner |
| `--verbose` | `-v` | `run` | Show logs to stderr |
| `--model` | `-m` | `run` | Override model |
| `--small-model` | | `run` | Override small model |

### No `--agent` Flag

Neither OpenCode nor Crush has a `--agent` flag. The concept of "agents" in both tools refers to internal agent types (coder, task, summarizer, title) -- not user-configurable agent personas.

---

## 7. Custom Agents and Skills

### OpenCode: No custom agent support

OpenCode has four hardcoded agent types:
- `coder` -- main coding agent with full tools
- `task` -- sub-agent spawned by coder (read-only tools: glob, grep, ls, view)
- `summarizer` -- summarizes conversations for context compaction
- `title` -- generates session titles

These cannot be customized beyond model selection.

### Crush: Agent Skills (agentskills.io standard)

Crush implements the Agent Skills specification (`SKILL.md` files):

**Discovery paths**:
1. `~/.config/crush/skills/` (default)
2. Configurable via `options.skills_paths` in config
3. Overridable via `CRUSH_SKILLS_DIR` env var

**SKILL.md format**:
```markdown
---
name: my-skill
description: Description of what this skill does
compatibility: Optional compatibility notes
---

Instructions for the agent when this skill is activated.
These instructions are injected into the system prompt.
```

Skills are discovered by walking directories recursively, looking for `SKILL.md` files. Valid skills are converted to XML and injected into the system prompt.

### Custom Commands (both tools)

Both support custom commands as markdown files:

**User commands**: `$XDG_CONFIG_HOME/crush/commands/*.md` (prefix: `user:`)
**Project commands**: `.crush/commands/*.md` (prefix: `project:`)

Commands support named arguments: `$PLACEHOLDER` in the markdown content.

In the TUI, custom commands are accessed via `Ctrl+K`.

---

## 8. Launching Programmatically in a Tmux Pane

### Recommended Approach for Cortex Integration

```bash
# OpenCode
opencode -c /path/to/project -p "Your kickoff prompt here" -q

# Crush
crush -c /path/to/project run "Your kickoff prompt here" --quiet --yolo
```

**Key considerations**:

1. **Working directory**: Use `-c` / `--cwd` flag to set the project root
2. **Config injection**: Write `.opencode.json` or `.crush.json` to the project root before launching, containing MCP server config pointing to the Cortex MCP
3. **Instructions injection**: Write an `opencode.md` or `AGENTS.md` file to the project root with the KICKOFF prompt / ticket context
4. **Permissions**: OpenCode auto-approves in `-p` mode. Crush needs `--yolo` flag for auto-approve
5. **Exit behavior**: Both tools exit after the prompt completes -- the tmux pane will close
6. **No session resume**: Cannot resume a previous session; each invocation is independent

### Config File Generation for MCP Injection

For Cortex to inject its MCP tools, generate a config file before spawning:

**OpenCode** (`.opencode.json`):
```json
{
  "mcpServers": {
    "cortex": {
      "type": "stdio",
      "command": "cortex-mcp-ticket",
      "env": [
        "CORTEX_PROJECT_PATH=/path/to/project",
        "CORTEX_TICKET_ID=abc123",
        "CORTEX_SESSION_ID=session-xyz"
      ],
      "args": []
    }
  },
  "contextPaths": ["opencode.md"],
  "autoCompact": true
}
```

**Crush** (`.crush.json`):
```json
{
  "mcp": {
    "cortex": {
      "type": "stdio",
      "command": "cortex-mcp-ticket",
      "env": {
        "CORTEX_PROJECT_PATH": "/path/to/project",
        "CORTEX_TICKET_ID": "abc123",
        "CORTEX_SESSION_ID": "session-xyz"
      },
      "timeout": 60
    }
  },
  "options": {
    "context_paths": ["AGENTS.md"]
  },
  "permissions": {
    "allowed_tools": ["view", "ls", "grep", "bash", "edit", "write", "patch"]
  }
}
```

---

## 9. Gaps and Limitations for Cortex Integration

### Critical Limitations

1. **No session resume/continue**: Cannot resume a previous session. Each `-p`/`run` invocation is independent. This means Cortex cannot implement "resume" mode for OpenCode/Crush agents.

2. **No `--system-prompt` flag**: Cannot pass arbitrary system prompts via CLI. Must use context files or provider-level `system_prompt_prefix`.

3. **No `--agent` flag**: Cannot specify which agent persona to use from CLI.

4. **Non-interactive mode exits on completion**: The process terminates when done. For interactive (TUI) mode, the process is long-running but cannot be controlled programmatically.

5. **MCP reconnects on every tool call** (OpenCode): Each tool invocation creates a new MCP client connection. This adds latency but shouldn't cause functional issues.

6. **No stdin prompt injection for interactive mode**: Cannot pipe a prompt into the interactive TUI -- it only works with the `run` subcommand (Crush) or `-p` flag (OpenCode).

### Differences from Claude Code Integration

| Feature | Claude Code | OpenCode/Crush |
|---------|------------|----------------|
| System prompt flag | `--system-prompt` | No (use context files) |
| Session resume | `--resume` flag | Not supported |
| MCP config | Via CLI flags or settings | Via config file only |
| Non-interactive | `-p` flag (stays alive) | `-p`/`run` (exits on done) |
| Custom instructions | `CLAUDE.md` | Multiple paths (auto-detected) |
| Agent selection | N/A (single agent) | N/A (hardcoded agents) |
| Headless operation | Well-supported | Supported but limited |

### Viable Integration Strategy

1. **Generate config files** before spawning (`.opencode.json` or `.crush.json`)
2. **Write instructions** to a context path file (e.g., `opencode.md` or `AGENTS.md`)
3. **Launch in interactive mode** in the tmux pane (not headless) so the agent runs as a TUI
4. **MCP tools** handle all Cortex communication (addComment, requestReview, concludeSession)
5. **Detect completion** via MCP tool calls (concludeSession triggers cleanup)
6. **No resume support** -- if the session is interrupted, must start fresh

For interactive mode in tmux, simply run `opencode -c /path/to/project` or `crush -c /path/to/project` and the TUI will start. The user (or architect) can then interact with it through the tmux pane. The context files and MCP config will be loaded automatically.

---

## 10. Agent Types in Cortex Config

The current Cortex config supports `opencode` as an agent type (see `internal/project/config/config.go`). The integration should:

1. Detect whether `opencode` or `crush` binary is available (Crush is the successor)
2. Generate the appropriate config file format
3. Write the context/instructions file
4. Launch the appropriate binary with the right flags
5. Monitor the tmux pane for session lifecycle events via MCP

The spawn launcher at `internal/core/spawn/launcher.go` would need OpenCode/Crush specific launch logic similar to what exists for Claude Code and Copilot.