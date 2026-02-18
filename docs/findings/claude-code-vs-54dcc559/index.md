---
id: 54dcc559-6e58-45f4-afd5-67a502754be8
title: Claude Code vs OpenCode Agent Defaults Comparison
tags:
    - agent-defaults
    - claude-code
    - opencode
    - comparison
    - research
created: 2026-02-13T10:27:59.448849Z
updated: 2026-02-13T10:27:59.448849Z
---
## Overview

This document compares the two agent default configurations shipped with Cortex: **Claude Code** (`internal/install/defaults/claude-code/`) and **OpenCode** (`internal/install/defaults/opencode/`). The analysis covers prompts, CLI arguments, MCP tool bindings, lifecycle hooks, and runtime spawning differences.

**Bottom line**: The prompts are 100% identical. All meaningful differences are in `cortex.yaml` configuration (args, permissions) and the runtime spawning layer (hooks, config delivery mechanism, CLI invocation).

---

## 1. Prompts — Identical

Every prompt file is **byte-for-byte identical** between Claude Code and OpenCode:

| Prompt Path | Identical? |
|-------------|------------|
| `architect/SYSTEM.md` | Yes |
| `architect/KICKOFF.md` | Yes |
| `meta/SYSTEM.md` | Yes |
| `meta/KICKOFF.md` | Yes |
| `ticket/work/SYSTEM.md` | Yes |
| `ticket/work/KICKOFF.md` | Yes |
| `ticket/work/APPROVE.md` | Yes |
| `ticket/debug/SYSTEM.md` | Yes |
| `ticket/debug/KICKOFF.md` | Yes |
| `ticket/debug/APPROVE.md` | Yes |
| `ticket/research/SYSTEM.md` | Yes |
| `ticket/research/KICKOFF.md` | Yes |
| `ticket/research/APPROVE.md` | Yes |
| `ticket/chore/SYSTEM.md` | Yes |
| `ticket/chore/KICKOFF.md` | Yes |
| `ticket/chore/APPROVE.md` | Yes |

Both have 16 prompt files with the same structure, same template variables, same workflow instructions.

---

## 2. cortex.yaml — Major Differences

This is where the two defaults diverge significantly.

### Claude Code (`claude-code/cortex.yaml`)

```yaml
architect:
  agent: claude
  args:
    - "--allow-dangerously-skip-permissions"
    - "--allowedTools"
    - "mcp__cortex__listTickets,mcp__cortex__readTicket"
meta:
  agent: claude
  args:
    - "--allow-dangerously-skip-permissions"
    - "--allowedTools"
    - "mcp__cortex__listProjects,mcp__cortex__readProjectConfig,..."
ticket:
  work:
    agent: claude
    args:
      - "--permission-mode"
      - "plan"
      - "--allow-dangerously-skip-permissions"
      - "--allowedTools"
      - "mcp__cortex__readReference"
  debug:
    agent: claude
    args:
      - "--permission-mode"
      - "plan"
      - "--allow-dangerously-skip-permissions"
      - "--allowedTools"
      - "mcp__cortex__readReference,mcp__cortex__addComment"
  research:
    agent: claude
    args:
      - "--allow-dangerously-skip-permissions"
      - "--allowedTools"
      - "mcp__cortex__readReference,mcp__cortex__addComment,mcp__cortex__createDoc"
  chore:
    agent: claude
    args:
      - "--allow-dangerously-skip-permissions"
      - "--allowedTools"
      - "mcp__cortex__readReference"
```

### OpenCode (`opencode/cortex.yaml`)

```yaml
architect:
  agent: opencode
meta:
  agent: opencode
ticket:
  work:
    agent: opencode
  debug:
    agent: opencode
  research:
    agent: opencode
  chore:
    agent: opencode
```

### Key Gaps in OpenCode Config

| Feature | Claude Code | OpenCode |
|---------|-------------|----------|
| **CLI args** | Fully specified per role | **None** — completely empty |
| **Permission mode** | `--permission-mode plan` for work/debug | Not configured |
| **Tool allowlisting** | `--allowedTools` scoped per role | Not configured |
| **Dangerous permissions bypass** | `--allow-dangerously-skip-permissions` | Not configured |

**Analysis**: Claude Code has carefully scoped permission controls — each session type gets exactly the MCP tools it needs pre-approved, while work/debug agents start in "plan" mode for extra safety. OpenCode has **zero** args configured, meaning it either relies on its own defaults or runs without equivalent guardrails.

---

## 3. MCP Tool Allowlisting (Claude Code only)

Claude Code's `--allowedTools` flag pre-approves specific Cortex MCP tools per session type, so the agent doesn't need user permission to call them:

| Session Type | Pre-approved Tools |
|--------------|-------------------|
| **Architect** | `listTickets`, `readTicket` |
| **Meta** | `listProjects`, `readProjectConfig`, `readGlobalConfig`, `daemonStatus`, `readDaemonLogs`, `listTickets`, `readTicket`, `listDocs`, `readDoc`, `listSessions`, `readPrompt` |
| **Ticket/work** | `readReference` |
| **Ticket/debug** | `readReference`, `addComment` |
| **Ticket/research** | `readReference`, `addComment`, `createDoc` |
| **Ticket/chore** | `readReference` |

**Note**: These are the *pre-approved* tools. All session types have access to more tools via MCP — the allowlist just controls which ones don't require user confirmation. Tools like `requestReview`, `concludeSession`, `addBlocker` are available but require user approval in Claude Code.

**OpenCode gap**: No equivalent allowlisting mechanism is configured. OpenCode's permission model (`"bypassPermissions"` mode with `"*": "allow"`) is set at runtime in the generated config, which is more permissive.

---

## 4. Runtime Spawning Differences

The spawning layer (`internal/core/spawn/`) treats the two agents very differently:

### Binary Invocation

| Aspect | Claude Code | OpenCode |
|--------|-------------|----------|
| Binary | `claude` | `opencode --agent cortex` |
| System prompt delivery | `--append-system-prompt` or `--system-prompt` flag | Embedded in `OPENCODE_CONFIG_CONTENT` env var |
| MCP config delivery | `--mcp-config <path>` (JSON file) | Embedded in `OPENCODE_CONFIG_CONTENT` env var |
| Settings/hooks delivery | `--settings <path>` (JSON file) | **Not supported** |
| Session resume | `--resume --session-id <id>` | Not documented |
| User prompt | `--prompt <content>` | `--prompt <content>` |

### Config Transformation

For OpenCode, Cortex transforms the Claude-format MCP config into OpenCode's format at runtime (`internal/core/spawn/opencode_config.go`):

- Claude format: `{ "mcpServers": { "name": { "command": "...", "args": [...], "env": {...} } } }`
- OpenCode format: `{ "mcp": { "name": { "type": "local", "command": ["cmd", "arg1", ..."], "environment": {...} } } }`

The system prompt is also embedded into the OpenCode config as a custom agent definition with `"mode": "bypassPermissions"`.

---

## 5. Lifecycle Hooks — Claude Code Only

Claude Code gets three lifecycle hooks injected via `--settings` (`internal/core/spawn/settings.go`):

| Hook | Trigger | Effect |
|------|---------|--------|
| `PostToolUse` | After any tool call | Updates agent status to "in_progress" |
| `Stop` | Agent stops | Updates agent status to "idle" |
| `PermissionRequest` | Agent requests permission | Updates agent status to "waiting_permission" |

These hooks call `cortexd hook <type>` which hits the `/agent/status` daemon endpoint, enabling the TUI to show real-time agent status.

**OpenCode gap**: No hooks are configured. The `GenerateSettingsConfig` function is explicitly skipped for OpenCode (`spawn.go:260-274`):

```go
if req.Agent != "opencode" {
    settingsConfig := GenerateSettingsConfig(...)
    ...
}
```

This means the TUI cannot track OpenCode agent status in real-time.

---

## 6. CONFIG_DOCS.md — Nearly Identical

Both have configuration documentation files. The only differences are:

- Claude Code mentions `--allowedTools` as a common customization example
- OpenCode omits `args` examples from its sample config (since none are set)
- Initialization command: `cortex init` vs `cortex init --agent opencode`

---

## 7. Summary of Gaps

### Critical Gaps (affect functionality)

1. **No CLI args** — OpenCode has zero args configured, meaning no permission scoping, no tool allowlisting, no plan mode
2. **No lifecycle hooks** — TUI can't track OpenCode agent status (in_progress, idle, waiting_permission)
3. **No session resume support** — OpenCode spawning doesn't appear to support `--resume`/`--session-id`

### Moderate Gaps (affect safety/UX)

4. **No tool allowlisting** — OpenCode agents get blanket `"*": "allow"` permissions via `bypassPermissions` mode rather than scoped per-tool approval
5. **No plan mode** — Work and debug tickets run without the extra "plan first" safety step

### Non-Gaps (parity achieved)

- Prompts: 100% identical
- MCP tool availability: Same tools registered per session type (Meta/Architect/Ticket)
- Template variables: Same set available
- Prompt resolution order: Same waterfall (project → extended base → embedded)
- Worktree support: Same config option

---

## 8. Recommendations

1. **Investigate OpenCode CLI args** — Does OpenCode support equivalent flags for permission scoping and tool allowlisting? If so, populate them in the defaults.
2. **OpenCode hooks** — If OpenCode supports any hook/callback mechanism, wire it up for agent status tracking. Without this, the TUI experience is degraded.
3. **Document the permission model difference** — Users choosing OpenCode should understand that it runs with broader permissions by default.
4. **Session resume** — Verify if OpenCode supports session resume and, if so, wire it into the launcher.