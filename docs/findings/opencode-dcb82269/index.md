---
id: dcb82269-dfd1-4fbf-9b5a-a83f84f20095
title: OpenCode Integration Points Research
tags:
    - opencode
    - integration
    - research
    - hooks
    - api
    - mcp
    - plugins
created: 2026-02-13T10:29:12.877895Z
updated: 2026-02-13T10:29:12.877895Z
---
## Summary

Comprehensive research into OpenCode's (github.com/opencode-ai/opencode) mechanisms for status reporting, lifecycle events, and integration hooks. OpenCode has evolved significantly and now provides rich integration capabilities through multiple channels: a plugin/hooks system, a full HTTP REST API with SSE events, an SDK, MCP client support, ACP (Agent Client Protocol), and a non-interactive CLI mode.

**Important note**: The original opencode-ai/opencode repository was archived on September 18, 2025. The project moved to anomalyco/opencode (previously sst/opencode) under the Charm team's stewardship. The active version is documented at opencode.ai.

---

## 1. Non-Interactive Mode, Status Reporting, and Machine-Readable Output

OpenCode provides robust non-interactive operation:

### CLI Command: `opencode run`
- Execute a single prompt non-interactively: `opencode run "prompt text"`
- `--format json` flag for machine-readable structured JSON output
- `--format default` for plain text output
- `-q` / `--quiet` flag suppresses spinner (useful for scripts)
- All permissions are auto-approved during non-interactive sessions
- `--attach` flag to connect to a running server instance
- `--file / -f` to attach files
- `--model / -m` to specify model
- `--agent` to select agent
- `--continue / -c` to continue a previous session
- `--session / -s` to specify session ID
- `--title` to set session title

### Other Machine-Readable Commands
- `opencode session list --format json` - JSON session listing
- `opencode export [sessionID]` - Full session data export as JSON
- `opencode models [provider] --verbose` - Model listing with costs metadata
- `opencode stats --days N` - Usage analytics

### Global Flags
- `--print-logs` - Output logs to stderr
- `--log-level DEBUG|INFO|WARN|ERROR` - Control log verbosity

---

## 2. API, WebSocket, and Event System

OpenCode has a full HTTP server with SSE (Server-Sent Events) for real-time monitoring.

### Server Mode: `opencode serve`
Runs a headless HTTP server with OpenAPI 3.1 spec at `/doc`.

```
opencode serve [--port 4096] [--hostname 127.0.0.1] [--mdns] [--cors origin]
```

Authentication via `OPENCODE_SERVER_PASSWORD` env var (HTTP basic auth, username "opencode").

### Key API Endpoints

| Category | Endpoint | Method | Description |
|----------|----------|--------|-------------|
| Health | `/global/health` | GET | Server health check |
| Events | `/global/event` | GET | SSE stream (all events) |
| Sessions | `/session` | GET/POST | List/create sessions |
| Sessions | `/session/:id` | GET/PATCH/DELETE | Read/update/delete session |
| Messages | `/session/:id/message` | GET/POST | List/send messages |
| Diffs | `/session/:id/diff` | GET | File diffs for session |
| Sharing | `/session/:id/share` | POST | Share session |
| Config | `/config` | GET/PATCH | Read/update configuration |
| Files | `/find` | GET | Search files |
| Tools | `/experimental/tool/ids` | GET | List available tools |
| Agents | `/agent` | GET | List available agents |
| TUI | `/tui/append-prompt` | POST | Drive TUI remotely |
| TUI | `/tui/submit-prompt` | POST | Submit prompt to TUI |
| Auth | `/auth/:id` | Various | Credential management |

### SSE Event Types

Events are streamed via `/global/event` and per-session endpoints:

**Session Events:**
- `session.created` - New session initialized
- `session.updated` - Session metadata changed
- `session.deleted` - Session removed
- `session.diff` - File diffs computed
- `session.error` - Processing failure
- `session.idle` - Agent finished responding (key for completion detection)

**Message Events:**
- `message.updated` - Message info changed
- `message.part.updated` - Streaming part updates with delta values
- `message.removed` - Message deleted
- `message.part.removed` - Part deleted

### Synchronous Message Endpoint
Critical finding: `POST /session/{id}/message` is **synchronous** -- the HTTP request blocks until the LLM finishes responding. This is the simplest way to detect session completion from external tools.

### SDK (TypeScript/JavaScript)
```
npm install @opencode-ai/sdk
```

```typescript
import { createOpencode } from "@opencode-ai/sdk"
const { client } = await createOpencode()

// Or connect to existing server:
const client = createOpencodeClient({ baseUrl: "http://localhost:4096" })

// Key operations:
client.session.create()
client.session.prompt({ path, body })
client.session.messages()
client.event.subscribe()  // SSE event stream
```

---

## 3. Hook/Callback Mechanism (Plugin System)

OpenCode has a rich plugin system with JavaScript/TypeScript hooks.

### Plugin Structure
Plugins are JS/TS modules in `.opencode/plugins/` that export functions:

```typescript
export async function myPlugin({ client, project, directory, worktree, $ }) {
  return {
    event: async ({ event }) => { /* handle events */ },
    stop: async ({ input }) => { /* intercept agent stop */ },
    tool: () => [{ /* custom tools */ }],
  }
}
```

### Available Hook Types

**Session Lifecycle:**
- `session.created` - New session started
- `session.updated` - Session state changed
- `session.compacted` - Context compaction occurred
- `session.idle` - Agent finished responding
- `session.error` - Error occurred
- `session.diff` - File changes detected
- `session.deleted` - Session removed

**Tool Execution:**
- `tool.execute.before` - Pre-execution (can modify/block)
- `tool.execute.after` - Post-execution (can react)

**File Operations:**
- `file.edited` - File modification events
- `file.watcher.updated` - Filesystem monitoring

**Message Lifecycle:**
- `message.updated` - Message changes
- `message.part.updated` - Component updates
- `message.removed` - Message deletion
- `message.part.removed` - Component removal

**Permissions:**
- `permission.asked` - Permission request
- `permission.replied` - Permission response

**System:**
- `shell.env` - Environment variable injection
- `command.executed` - Command completion
- `installation.updated` - Dependency changes
- `server.connected` - Server connectivity
- `todo.updated` - Task state changes

**UI/Notifications:**
- `tui.prompt.append` - Prompt events
- `tui.command.execute` - Command execution
- `tui.toast.show` - Toast notifications

**Advanced:**
- `experimental.chat.system.transform` - Inject into system prompts
- `experimental.session.compacting` - Custom compaction behavior
- `stop` hook - Intercept agent termination

### Plugin Dependencies
Plugins can use npm packages via `.opencode/package.json`.

---

## 4. Logging and File-Based State

### Log Files
- **Location**: `~/.local/share/opencode/log/`
- **Format**: Timestamped files (e.g., `2025-01-09T123456.log`)
- **Retention**: 10 most recent files kept
- **Control**: `--log-level DEBUG` for verbose output, `--print-logs` for stderr

### Data Directory Structure
```
~/.local/share/opencode/
├── auth.json              # API keys and OAuth tokens
├── mcp-auth.json          # MCP OAuth tokens
├── log/                   # Application logs
└── project/
    ├── <project-slug>/    # Git repo sessions
    │   └── storage/       # Session and message data
    └── global/            # Non-git repo storage
        └── storage/
```

### Configuration Locations
```
~/.config/opencode/opencode.jsonc    # Global config
<project>/.opencode/                  # Project-specific
<project>/opencode.json               # Project root config
```

### Cache
- `~/.cache/opencode` - Plugin and provider package cache

### SQLite Database
OpenCode uses SQLite for persistent storage of sessions, messages, and file history. The database is stored within the project storage directory.

---

## 5. MCP Server/Client Integration

### OpenCode as MCP Client
OpenCode can connect to MCP servers defined in configuration:

```json
{
  "mcp": {
    "cortex": {
      "type": "local",
      "command": ["cortexd", "mcp", "ticket"],
      "environment": {
        "CORTEX_PROJECT_PATH": "/path/to/project"
      },
      "timeout": 5000,
      "enabled": true
    }
  }
}
```

**Local servers** use stdio protocol. **Remote servers** use HTTP with optional OAuth.

### Tool Management
- MCP tools appear as standard tools alongside built-in tools
- Can be enabled/disabled globally or per-agent via glob patterns
- Permission system applies: `"mymcp_*": "ask"`

### Configuration Injection for MCP
`OPENCODE_CONFIG_CONTENT` env var allows injecting MCP configuration:
```
OPENCODE_CONFIG_CONTENT='{"mcp":{"cortex":{"type":"local","command":["cortexd","mcp","ticket"]}}}'
```

Combined with `OPENCODE_DISABLE_PROJECT_CONFIG=true` for clean injection.

### ACP (Agent Client Protocol)
OpenCode supports ACP for editor integration:
```
opencode acp [--cwd dir] [--port port] [--hostname host]
```
Used by Zed, Avante.nvim, and other ACP-compatible editors.

---

## 6. Permission/Input Waiting Behavior

### Permission System
Tools can be configured with three permission states:
- `"allow"` - Auto-approve
- `"deny"` - Block
- `"ask"` - Require user approval

### In Non-Interactive Mode
All permissions are auto-approved during `opencode run` sessions.

### In TUI Mode
When waiting for user input on permissions:
- `a` - Allow single action
- `A` - Allow for session
- `d` - Deny

### Programmatic Permission Control
The `permission.asked` and `permission.replied` plugin hooks can intercept permission requests. The server API exposes permission state through session events.

### Bypass Mode
Agent configuration can set `"permission": {"*": "allow"}` to bypass all permission prompts. Cortex currently uses this approach (mode: "bypassPermissions").

---

## 7. Environment Variables and Configuration for External Integration

### Key Environment Variables

| Variable | Purpose |
|----------|---------|
| `OPENCODE_CONFIG_CONTENT` | Inject full config as JSON |
| `OPENCODE_DISABLE_PROJECT_CONFIG` | Ignore project-level config |
| `OPENCODE_CONFIG_DIR` | Custom config directory |
| `OPENCODE_SERVER_PASSWORD` | HTTP basic auth password |
| `OPENCODE_ENABLE_EXA` | Enable web search tool |
| `ANTHROPIC_API_KEY` | Anthropic provider |
| `OPENAI_API_KEY` | OpenAI provider |
| `GITHUB_TOKEN` | GitHub integration |
| `SHELL` | Default shell for bash tool |
| `LOCAL_ENDPOINT` | Self-hosted model endpoint |

### Config Variable Substitution
Config files support:
- `{env:VARIABLE_NAME}` - Environment variable substitution
- `{file:path/to/file}` - File content substitution

### Remote Config
Organizations can host defaults at `.well-known/opencode` endpoints, which local configs can override.

### Config Schema
Schema validation via `"$schema": "https://opencode.ai/config.json"` for IDE autocomplete.

---

## Implications for Cortex Integration

### Current Cortex OpenCode Support
Cortex currently integrates with OpenCode via:
1. `OPENCODE_CONFIG_CONTENT` env var to inject MCP config and agent definition
2. `--agent cortex` flag to select the injected agent
3. `--prompt` flag to pass the kickoff message
4. `bypassPermissions` mode for auto-approval

### Potential Enhancement Opportunities

1. **Server Mode Integration**: Instead of spawning OpenCode in TUI mode within tmux, Cortex could spawn `opencode serve` and interact via HTTP API. This would enable:
   - Programmatic session state monitoring via SSE
   - Detecting when agent is idle/done via `session.idle` events
   - Sending follow-up prompts without tmux send-keys
   - Getting structured output via `--format json`

2. **Plugin-Based Integration**: A Cortex plugin for OpenCode could:
   - Report `session.idle` events back to Cortex daemon
   - Track file changes and tool executions
   - Enforce workflow requirements via the `stop` hook
   - Inject Cortex context via `shell.env` hook

3. **Non-Interactive Orchestration**: Using `opencode run --format json --attach` with a running server could provide clean programmatic control.

4. **SDK Integration**: The `@opencode-ai/sdk` TypeScript SDK could be used in a sidecar process to monitor and control OpenCode sessions.

5. **SSE Event Monitoring**: Subscribing to `/global/event` SSE stream would provide real-time session state updates without polling.

6. **Synchronous Message API**: `POST /session/{id}/message` blocks until completion -- ideal for simple orchestration workflows.