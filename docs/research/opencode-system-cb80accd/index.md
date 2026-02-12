---
id: cb80accd-238c-48b6-9346-7e9f64dc7162
title: 'OpenCode System Prompt Injection: Complete Guide'
tags:
    - opencode
    - system-prompt
    - configuration
    - agents
    - prompt-injection
created: 2026-02-11T08:40:26.538759Z
updated: 2026-02-11T08:40:26.538759Z
---
## Summary

OpenCode has **no `--system-prompt` CLI flag** and **no `SYSTEM.md` convention**. However, it provides 6 distinct methods for injecting custom system prompts, ranging from simple file drops to programmatic plugin hooks. The key insight is understanding what gets **replaced** vs. **appended** — the agent prompt replaces the provider-specific base prompt, but environment info and instruction files are always added alongside it.

---

## How the System Prompt is Assembled

The final system prompt sent to the LLM is assembled in `packages/opencode/src/session/llm.ts:66-96`:

```
┌─────────────────────────────────────────┐
│  1. Agent prompt  OR  Provider prompt   │  ← REPLACEABLE via agent config
├─────────────────────────────────────────┤
│  2. Environment info                    │  ← ALWAYS added (model, cwd, platform, date)
├─────────────────────────────────────────┤
│  3. Instruction files                   │  ← ALWAYS added (AGENTS.md, CLAUDE.md, etc.)
├─────────────────────────────────────────┤
│  4. User-message system override        │  ← Per-message (internal API only)
├─────────────────────────────────────────┤
│  5. Plugin transform                    │  ← Can modify/replace entire system array
└─────────────────────────────────────────┘
```

**Key code** (`llm.ts:66-79`):
```typescript
const system = []
system.push([
  // Agent prompt replaces provider prompt entirely
  ...(input.agent.prompt ? [input.agent.prompt] : SystemPrompt.provider(input.model)),
  // Environment + instruction files (from prompt.ts)
  ...input.system,
  // Per-message system override
  ...(input.user.system ? [input.user.system] : []),
].filter((x) => x).join("\n"))
```

The `input.system` is built in `prompt.ts:627`:
```typescript
system: [
  ...(await SystemPrompt.environment(model)),   // model ID, cwd, platform, date
  ...(await InstructionPrompt.system())          // AGENTS.md, CLAUDE.md, instructions config
]
```

---

## Injection Method 1: Agent Markdown File (Recommended)

**Best for: Full system prompt replacement with file-based management**

Create `.opencode/agents/<name>.md` (or `.opencode/agent/<name>.md`):

```markdown
---
description: My custom agent with full prompt control
mode: primary
model: anthropic/claude-sonnet-4-20250514
permission:
  "*": allow
---

Your complete system prompt goes here.
This replaces the provider-specific base prompt entirely.
It supports full markdown formatting.
```

The markdown body becomes `agent.prompt`, which **replaces** the provider-specific prompt (e.g., the Anthropic prompt, Gemini prompt, etc.). The filename (minus `.md`) becomes the agent name.

**Usage:**
```bash
opencode --agent <name>
opencode run --agent <name> "your message"
```

**Source:** `config/config.ts:379-408` — `loadAgent()` reads `{agent,agents}/**/*.md`, parses frontmatter via `gray-matter`, sets `prompt: md.content.trim()`.

**Caveat:** Environment info and instruction files (AGENTS.md etc.) are still appended to the system prompt alongside the agent prompt.

---

## Injection Method 2: JSON Config `prompt` Field

**Best for: Inline prompt definition or file references via `{file:path}` substitution**

In `opencode.json` or `opencode.jsonc`:

```jsonc
{
  "agent": {
    "myagent": {
      "description": "Custom agent",
      "mode": "primary",
      "prompt": "You are a specialized agent that..."
    }
  }
}
```

Or with file reference (variable substitution runs on the raw JSON text before parsing):

```jsonc
{
  "agent": {
    "myagent": {
      "description": "Custom agent",
      "mode": "primary",
      "prompt": "{file:./prompts/system-prompt.txt}"
    }
  }
}
```

Or with environment variable:

```jsonc
{
  "agent": {
    "myagent": {
      "prompt": "{env:MY_SYSTEM_PROMPT}"
    }
  }
}
```

**Source:** `config/config.ts:1244-1265` — `{env:VAR}` and `{file:path}` substitution runs on the raw config text before JSON parsing. Works for ALL fields including `prompt`.

---

## Injection Method 3: `OPENCODE_CONFIG_CONTENT` Environment Variable

**Best for: CI/CD, scripted one-off runs, dynamic prompt injection**

```bash
OPENCODE_CONFIG_CONTENT='{"agent":{"myagent":{"description":"Custom","mode":"primary","prompt":"You are a..."}}}' \
  opencode run --agent myagent "do something"
```

**Important caveat:** `OPENCODE_CONFIG_CONTENT` uses `JSON.parse()` directly (`config.ts:179`) — it does NOT go through the `load()` function. This means `{file:path}` and `{env:VAR}` substitution do NOT work. Only literal JSON strings are supported. You must inline the full prompt text.

**Priority:** This overrides all config sources except managed (enterprise) config. It is the highest-priority non-enterprise config source.

---

## Injection Method 4: `instructions` Config Field

**Best for: Adding supplementary instructions without replacing the base prompt**

In `opencode.json`:

```jsonc
{
  "instructions": [
    "./my-rules.md",
    "./prompts/*.md",
    "**/.cursorrules",
    "https://example.com/instructions.txt"
  ]
}
```

**Key difference:** This does NOT replace the provider prompt. These files are loaded by `InstructionPrompt.system()` and appended to the system prompt alongside the provider/agent prompt. Supports:
- Relative file paths (resolved via `findUp` from cwd to project root)
- Absolute file paths
- Glob patterns
- HTTP/HTTPS URLs (fetched with 5s timeout)

**Source:** `instruction.ts:94-113` — iterates `config.instructions`, resolves paths/globs, reads file contents.

---

## Injection Method 5: Context Files (AGENTS.md, CLAUDE.md, CONTEXT.md)

**Best for: Simple instruction injection compatible with other tools**

Drop a file in the project directory tree:

| File | Purpose |
|------|---------|
| `AGENTS.md` | Primary (preferred by OpenCode) |
| `CLAUDE.md` | Claude Code compatibility |
| `CONTEXT.md` | Deprecated, still supported |

**Search behavior:** Uses `findUp` from cwd to project root. First filename match wins (checked in order: AGENTS.md → CLAUDE.md → CONTEXT.md). Only one file per directory level is loaded.

**Global files** (also loaded):
- `~/.config/opencode/AGENTS.md`
- `~/.claude/CLAUDE.md` (unless `OPENCODE_DISABLE_CLAUDE_CODE_PROMPT=1`)

**Key difference:** Like `instructions`, these are appended — they do NOT replace the provider/agent prompt.

**Source:** `instruction.ts:71-116` — `systemPaths()` collects all paths, `system()` reads and returns their contents with `"Instructions from: <path>"` headers.

---

## Injection Method 6: Plugin System Transform Hook

**Best for: Programmatic, runtime modification of the entire system prompt**

Create a plugin (`.opencode/plugin/my-plugin.ts` or install via npm):

```typescript
import type { Plugin } from "opencode-ai"

export default {
  name: "my-prompt-plugin",
  hooks: {
    "experimental.chat.system.transform": async (context, data) => {
      // data.system is a mutable string[]
      // Replace the entire system prompt:
      data.system.length = 0
      data.system.push("Your completely custom system prompt")
      
      // Or append to it:
      data.system.push("Additional instructions from plugin")
      
      // Context provides: sessionID, model
    }
  }
}
```

**Source:** `llm.ts:83-90` — triggers `experimental.chat.system.transform` hook, passes mutable `{ system }` object. If the hook empties the array, the original system prompt is restored (safety fallback).

**Power:** This is the most powerful method — it can completely replace, modify, or extend the system prompt at runtime with full programmatic control.

---

## Comparison Matrix

| Method | Replaces Base Prompt? | Env Info Added? | Instructions Added? | `{file:}` / `{env:}` Support | Best For |
|--------|----------------------|-----------------|--------------------|-----------------------------|----------|
| Agent `.md` file | Yes | Yes | Yes | N/A (body is the prompt) | Full prompt control, file-managed |
| JSON `prompt` field | Yes | Yes | Yes | Yes | Dynamic file/env references |
| `OPENCODE_CONFIG_CONTENT` | Yes | Yes | Yes | No (literal JSON only) | CI/CD, scripted runs |
| `instructions` config | No (appended) | Yes | Yes (self) | Yes (in config file) | Supplementary rules |
| Context files (AGENTS.md) | No (appended) | Yes | Yes (self) | N/A | Cross-tool compatibility |
| Plugin transform hook | Full control | Depends | Depends | N/A (programmatic) | Runtime modification |

---

## Achieving a "Clean" System Prompt

To get the closest to a pure, controlled system prompt:

1. **Define a custom agent** with your prompt as the body (Method 1 or 2) — this replaces the provider-specific base prompt
2. **Set `OPENCODE_DISABLE_CLAUDE_CODE_PROMPT=1`** — prevents loading `~/.claude/CLAUDE.md`
3. **Don't place** `AGENTS.md`, `CLAUDE.md`, or `CONTEXT.md` in the project or global directories
4. **Don't configure** `instructions` in config
5. **Environment info is always added** — you cannot suppress the model ID, working directory, platform, and date block (`SystemPrompt.environment()` in `system.ts:29-53`)

The only way to fully suppress environment info is via a **plugin transform hook** (Method 6) that clears and replaces the entire system array.

---

## Cortex Integration Recommendation

For Cortex's use case (spawning OpenCode as a ticket agent), the recommended approach is:

```bash
# Method A: Agent markdown file (static, file-managed)
# Create .opencode/agents/cortex-ticket.md with the Cortex system prompt as body

# Method B: JSON config with file reference (dynamic, reusable)
# In opencode.json:
# { "agent": { "cortex-ticket": { "prompt": "{file:.cortex/prompts/ticket-agent.txt}" } } }

# Method C: Inline via env var (ephemeral, per-invocation)
OPENCODE_CONFIG_CONTENT='{"agent":{"cortex":{"mode":"primary","prompt":"You are a Cortex ticket agent..."}}}' \
  opencode run --agent cortex "work on this ticket"
```

**Method A** is simplest for persistent configuration. **Method C** is best for dynamic, per-ticket prompt injection where the prompt changes per invocation.