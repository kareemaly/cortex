---
id: 855a91d5-f01a-49e1-8664-9881655d4842
title: Use OpenCode instruction files for ticket agent SYSTEM.md instead of agent.prompt
type: work
tags:
    - opencode
    - agents
references:
    - doc:324cb419-80fe-4c49-929f-12e22bd0e9e6
created: 2026-02-14T12:20:01.96249Z
updated: 2026-02-14T12:28:26.390743Z
---
## Problem

When Cortex spawns OpenCode ticket agent sessions, it sets `agent.cortex.prompt` in `OPENCODE_CONFIG_CONTENT`, which **replaces** OpenCode's built-in provider prompt (`anthropic.txt` — ~106 lines of behavioral guidelines). This causes OpenCode ticket agents to lose all default tool usage policies, output formatting, and coding behavior guidance.

For Claude Code ticket agents, `--append-system-prompt` preserves the built-in prompt. OpenCode ticket agents should have equivalent behavior.

**Note:** For architect and meta sessions, replacing the system prompt is intentional. This change is **only for ticket agent sessions**.

## Solution

Use OpenCode's `instructions` config field instead of `agent.prompt`:

1. **Stop setting `agent.cortex.prompt`** for ticket agents — so OpenCode keeps its built-in `anthropic.txt` provider prompt
2. **Write the resolved SYSTEM.md to a temp file** (e.g., `/tmp/cortex-{ticket-id}-system.md`)
3. **Add that temp file path to the `instructions` array** in `OPENCODE_CONFIG_CONTENT`

OpenCode's config merging concatenates instruction arrays (`mergeConfigConcatArrays`), so this won't conflict with project-level AGENTS.md or CLAUDE.md files. Each instruction file is wrapped with `Instructions from: /path` and appended to the system prompt.

The resulting prompt assembly for ticket agents:
```
[Environment info]
[Project AGENTS.md / CLAUDE.md]
[Cortex SYSTEM.md via instructions]
[Anthropic provider prompt - anthropic.txt]
```

## Scope

- Only change behavior for **ticket agent** sessions when agent type is OpenCode
- Architect and meta sessions should continue using `agent.prompt` (intentional replacement)
- The temp file should be cleaned up when the session concludes

## References

- doc:324cb419 — OpenCode System Prompt: Append vs Replace Analysis

## Acceptance Criteria

- OpenCode ticket agents preserve the built-in `anthropic.txt` provider prompt
- Cortex's SYSTEM.md content is injected via the `instructions` config field
- Architect and meta OpenCode sessions are unchanged (still replace system prompt)
- Temp instruction files are created and cleaned up appropriately
- Build passes, tests pass