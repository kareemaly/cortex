---
id: b944b83c-a9dd-43f9-825c-e1cd37a23800
title: Codex CLI Integration with Cortex — Research Findings
tags:
    - codex
    - integration
    - agent
    - openai
    - mcp
created: 2026-02-11T08:01:32.872823Z
updated: 2026-02-11T08:01:32.872823Z
---
## Executive Summary

**Recommendation: Yes, integrate Codex as a supported agent type.** The integration is feasible and follows the established Copilot pattern (no system prompt flag, no hooks). The primary challenge is MCP server injection — Codex uses `codex mcp add/remove` CLI commands instead of a `--mcp-config <file>` flag, requiring a launcher script wrapper approach.

Codex CLI (v0.77.0, `@openai/codex`) is a mature agentic coding tool with interactive TUI mode, headless `exec` mode, full MCP support, session persistence/resume, sandbox isolation, and configurable approval policies. It maps well to Cortex's agent lifecycle.

---

## 1. Codex CLI Capabilities

### Relevant Features

| Feature | Details |
|---------|---------|
| **Interactive mode** | `codex [PROMPT]` — TUI session, suitable for tmux panes |
| **Headless mode** | `codex exec [PROMPT]` — non-interactive, exits on completion |
| **MCP support** | `codex mcp add/remove/list` — stdio and HTTP transports |
| **Session resume** | `codex resume [SESSION_ID]` / `codex exec resume` |
| **Sandbox modes** | `read-only`, `workspace-write`, `danger-full-access` |
| **Approval policies** | `untrusted`, `on-failure`, `on-request`, `never` |
| **Full-auto mode** | `--full-auto` = `-a on-request --sandbox workspace-write` |
| **Working directory** | `-C, --cd <DIR>` — set agent working directory |
| **Model selection** | `-m, --model <MODEL>` — override default model |
| **Config profiles** | `-p, --profile <NAME>` — named config profiles |
| **Stdin prompt** | `codex exec -` — read prompt from stdin |
| **Config override** | `-c key=value` — runtime TOML config overrides |
| **Additional dirs** | `--add-dir <DIR>` — make extra directories writable |

### What Codex Lacks (vs Claude Code)

| Missing Feature | Impact on Cortex |
|----------------|-----------------|
| No `--system-prompt` flag | Must embed instructions in KICKOFF.md (same as Copilot) |
| No `--mcp-config <file>` flag | Must use `codex mcp add/remove` in launcher script |
| No `--settings` for hooks | No lifecycle hooks (same as Copilot) |
| No `--session-id` flag | Cannot assign specific session IDs |
| No per-invocation MCP config | MCP servers registered globally in `~/.codex/config.toml` |

### Instruction Injection Mechanisms

Codex reads instructions from:
1. `~/.codex/instructions.md` — global (always loaded)
2. `CODEX.md` in project root — project-scoped
3. The `[PROMPT]` CLI argument — per-invocation
4. `-c key=value` config overrides — per-invocation

Since there's no `--system-prompt` flag, all Cortex workflow instructions must go into the prompt (KICKOFF.md), matching the established Copilot pattern.

---

## 2. Integration Architecture

### Comparison with Existing Agents

| Aspect | Claude Code | Copilot | Codex (proposed) |
|--------|-------------|---------|-------------------|
| **Command** | `claude` | `gh copilot agent` | `codex` |
| **MCP flag** | `--mcp-config <file>` | `--additional-mcp-config <file>` | `codex mcp add/remove` (in launcher) |
| **System prompt** | `--system-prompt` / `--append-system-prompt` | Not supported | Not supported |
| **Settings/hooks** | `--settings <file>` | Not supported | Not supported |
| **Session ID** | `--session-id <uuid>` | Not supported | Not supported |
| **Resume** | `--resume [id]` | `--resume` | `codex resume --last` |
| **Automation** | `--allow-dangerously-skip-permissions` + `--allowedTools` | `--yolo` | `--full-auto` or `--dangerously-bypass-approvals-and-sandbox` |
| **Prompt delivery** | SYSTEM.md + KICKOFF.md | KICKOFF.md only | KICKOFF.md only |
| **Working dir** | Inherits from tmux | Inherits from tmux | `-C <dir>` or inherits |

### Proposed Agent Type

Codex follows the **Copilot pattern** — no system prompt, no hooks, instructions embedded in KICKOFF.md.

---

## 3. MCP Injection — The Core Challenge

This is the most significant integration difference. Claude and Copilot both accept an MCP config JSON file path as a CLI flag. Codex manages MCP servers via `codex mcp add/remove`, which persists config in `~/.codex/config.toml`.

### Option A: Launcher Script Add/Remove (Recommended)

Wrap MCP registration in the launcher script:

```bash
#!/usr/bin/env bash
trap 'codex mcp remove cortex; rm -f /tmp/cortex-*.txt' EXIT

export CORTEX_TICKET_ID="abc123"
export CORTEX_PROJECT="/path/to/project"

# Register MCP server for this session
codex mcp add cortex \
  --env CORTEX_PROJECT_PATH="/path/to/project" \
  --env CORTEX_DAEMON_URL="http://127.0.0.1:4200" \
  --env CORTEX_TMUX_SESSION="myproject" \
  -- /path/to/cortexd mcp --ticket-id abc123 --ticket-type work

# Launch agent
codex --full-auto "$(cat /tmp/cortex-prompt.txt)"
```

**Pros:**
- Uses Codex's native MCP management
- Clean registration/cleanup via trap
- Same MCP server binary and protocol as other agents

**Cons:**
- Global config (`~/.codex/config.toml`) — potential conflicts with concurrent sessions
- Must ensure cleanup even on crashes (trap handles normal exits)
- Only one Codex agent can use the "cortex" MCP name at a time

### Option B: Unique MCP Names Per Session

```bash
MCP_NAME="cortex-${CORTEX_TICKET_ID}"
codex mcp add "$MCP_NAME" --env ... -- cortexd mcp --ticket-id ...
codex --full-auto "$(cat prompt.txt)"
trap 'codex mcp remove "$MCP_NAME"' EXIT
```

**Pros:** Supports concurrent Codex sessions
**Cons:** Tool names become `cortex-abc123__readReference` instead of `cortex__readReference`, breaking prompt templates

### Option C: CODEX.md with Embedded Instructions

Write a temporary `CODEX.md` in the working directory that tells the agent to use cortexd tools.

**Verdict:** Rejected — MCP tools need structured registration, not just text instructions.

### Option D: Config Profile Per Session

Use `-p <profile>` with per-session MCP configs in config.toml profiles.

**Verdict:** Needs testing — unclear if profiles support MCP server definitions.

### Recommendation

**Option A** for initial implementation. Accept the constraint of one Codex agent per machine. If concurrent Codex sessions become a priority, investigate Option D (profiles) or lobby OpenAI for a `--mcp-config <file>` flag.

---

## 4. Implementation Plan

### Files to Modify

| File | Change |
|------|--------|
| `internal/project/config/config.go` | Add `AgentCodex AgentType = "codex"`, update `Validate()` |
| `internal/core/spawn/launcher.go` | Add `buildCodexCommand()` and `buildCodexMCPSetup()` |
| `internal/core/spawn/spawn.go` | Skip settings config for codex (like copilot), handle MCP differently |
| `internal/core/spawn/config.go` | Generate MCP add/remove commands instead of JSON file for codex |

### Files to Create

| File | Purpose |
|------|---------|
| `internal/install/defaults/codex/cortex.yaml` | Default agent config |
| `internal/install/defaults/codex/CONFIG_DOCS.md` | Configuration docs |
| `internal/install/defaults/codex/prompts/architect/KICKOFF.md` | Architect prompt (instructions + ticket list) |
| `internal/install/defaults/codex/prompts/ticket/work/KICKOFF.md` | Work ticket prompt |
| `internal/install/defaults/codex/prompts/ticket/work/APPROVE.md` | Approval instructions |
| `internal/install/defaults/codex/prompts/ticket/debug/KICKOFF.md` | Debug ticket prompt |
| `internal/install/defaults/codex/prompts/ticket/debug/APPROVE.md` | Debug approval |
| `internal/install/defaults/codex/prompts/ticket/research/KICKOFF.md` | Research prompt |
| `internal/install/defaults/codex/prompts/ticket/research/APPROVE.md` | Research approval |
| `internal/install/defaults/codex/prompts/ticket/chore/KICKOFF.md` | Chore prompt |
| `internal/install/defaults/codex/prompts/ticket/chore/APPROVE.md` | Chore approval |

### Proposed Default Config

```yaml
# ~/.cortex/defaults/codex/cortex.yaml
architect:
  agent: codex
  args:
    - "--full-auto"
ticket:
  work:
    agent: codex
    args:
      - "--full-auto"
  debug:
    agent: codex
    args:
      - "--full-auto"
  research:
    agent: codex
    args:
      - "--full-auto"
      - "-s"
      - "read-only"
  chore:
    agent: codex
    args:
      - "--full-auto"
git:
  worktrees: false
```

Note: research tickets use `-s read-only` sandbox to enforce read-only mode.

### Proposed Launcher Command

```bash
buildCodexCommand(params LauncherParams) string:

# MCP setup (before main command)
codex mcp add cortex \
  --env CORTEX_PROJECT_PATH=<project> \
  --env CORTEX_DAEMON_URL=<url> \
  --env CORTEX_TMUX_SESSION=<session> \
  -- <cortexd-path> mcp --ticket-id <id> --ticket-type <type>

# Main command
codex "$(cat <prompt-file>)" \
  [--full-auto | -s <sandbox-mode>] \
  [extra args from config]
```

For resume:
```bash
codex resume --last
```

### Launcher Script Structure

The launcher script needs a different structure than Claude/Copilot because MCP setup happens before the command, not as a flag:

```bash
#!/usr/bin/env bash
trap 'codex mcp remove cortex; rm -f <temp-files>' EXIT

export CORTEX_TICKET_ID="<id>"
export CORTEX_PROJECT="<path>"

# Register Cortex MCP server
codex mcp add cortex \
  --env CORTEX_PROJECT_PATH='<project-path>' \
  --env CORTEX_DAEMON_URL='http://127.0.0.1:4200' \
  --env CORTEX_TMUX_SESSION='<session>' \
  -- '<cortexd-path>' mcp --ticket-id '<id>' --ticket-type '<type>'

# Launch agent
codex --full-auto "$(cat '<prompt-file>')" <extra-args>
```

This means `buildLauncherScript()` in `launcher.go` needs to be refactored slightly — the codex case needs to emit MCP setup commands before the main agent command, not as part of the command itself.

---

## 5. Prompt Templates

Follow the Copilot convention: no SYSTEM.md files, all instructions embedded in KICKOFF.md.

### Key Differences from Copilot Prompts

1. **MCP tool prefix**: Codex may use `cortex__` instead of `mcp__cortex__` depending on how it formats MCP tool names. This needs to be verified by testing.
2. **Sandbox awareness**: Prompts can reference sandbox modes (`read-only` for research).
3. **CODEX.md reference**: Prompts should not conflict with any user-managed CODEX.md.

---

## 6. Session Lifecycle

| Stage | Codex Behavior | Cortex Handling |
|-------|---------------|-----------------|
| **Spawn** | `codex --full-auto "prompt"` | Register MCP, launch in tmux |
| **Working** | Agent uses MCP tools (addComment, readReference, etc.) | Same as Claude/Copilot |
| **Review** | Agent calls `requestReview` | Ticket moves to review |
| **Approve** | User approves; agent gets APPROVE.md content | Same as Claude/Copilot |
| **Conclude** | Agent calls `concludeSession` | MCP removed via trap, ticket to done |
| **Resume** | `codex resume --last` | Re-register MCP, launch with resume |
| **Orphaned** | Codex TUI closed but session persists | Detect via tmux, offer resume |

### Session State Detection

Codex stores sessions in `~/.codex/sessions/` by date. However, Cortex tracks sessions in `.cortex/sessions.json` independently. For state detection:
- **tmux window exists** = active session
- **tmux window gone + cortex session record exists** = orphaned
- **No record** = normal (ready to spawn)

This is the same as Claude/Copilot — Cortex doesn't rely on the agent's internal session store.

---

## 7. Gaps and Blockers

### Blocking Issues

| Issue | Severity | Mitigation |
|-------|----------|------------|
| No `--mcp-config <file>` flag | High | Use `codex mcp add/remove` in launcher script |
| Concurrent session MCP conflicts | Medium | Accept one-at-a-time constraint initially |

### Non-Blocking Gaps

| Gap | Impact | Notes |
|-----|--------|-------|
| No `--system-prompt` | Low | Use Copilot pattern (KICKOFF.md) |
| No `--settings` hooks | Low | Same as Copilot |
| No `--session-id` | Low | Cortex tracks sessions independently |
| MCP tool name prefix unknown | Low | Test to confirm; may need template adjustment |
| Global MCP config | Medium | Cleanup needed; investigate profiles later |

### Risks

1. **MCP cleanup on crash**: If the agent process or machine crashes without hitting the trap, the MCP server registration persists in `~/.codex/config.toml`. This is recoverable (just stale config) but could cause confusion on next spawn. Solution: check for and clean up stale registrations before each spawn.

2. **Concurrent Codex agents**: Unlike Claude/Copilot where each session gets its own MCP config file, Codex shares a global config. Running multiple Codex ticket agents simultaneously would require unique MCP names, which changes tool name prefixes. This is a significant limitation.

3. **Codex version churn**: Codex CLI is actively developed (v0.77.0 local vs v0.98.0 on npm). API stability is not guaranteed. Flag names and MCP behavior could change.

---

## 8. Implementation Complexity Estimate

Following the Copilot integration as a template:

| Component | Effort | Notes |
|-----------|--------|-------|
| Config + validation | Small | Add constant, update validation |
| Launcher command builder | Medium | New builder + MCP setup logic |
| Default config + prompts | Medium | ~12 files, follow copilot pattern |
| Spawn logic updates | Small | Skip settings (like copilot) |
| Launcher script refactor | Medium | Pre-command MCP setup + cleanup |
| Testing | Medium | Unit tests + manual tmux validation |
| CONFIG_DOCS.md | Small | Documentation |

---

## 9. Open Questions

1. **What prefix does Codex use for MCP tools?** Claude uses `mcp__<server>__<tool>`. Does Codex use the same convention? This affects all prompt templates.

2. **Do config profiles support MCP server definitions?** If `-p <profile>` can scope MCP servers, this solves the concurrency problem.

3. **Can `-c` override MCP config at runtime?** If `-c 'mcp_servers.cortex.command=...'` works, we might not need `mcp add/remove` at all.

4. **Does Codex support project-level `.codex/` directories?** Project-scoped config would isolate MCP settings per workspace.

5. **Does `codex exec` mode work well enough for autonomous agents?** Or is interactive mode (`codex --full-auto`) better for the tmux use case?

These questions should be answered via hands-on testing before implementation begins.