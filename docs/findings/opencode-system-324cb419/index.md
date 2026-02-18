---
id: 324cb419-80fe-4c49-929f-12e22bd0e9e6
title: 'OpenCode System Prompt: Append vs Replace Analysis'
tags:
    - opencode
    - system-prompt
    - spawn
    - architecture
created: 2026-02-14T11:46:33.452938Z
updated: 2026-02-14T11:46:33.452938Z
---
## TL;DR

**The Cortex SYSTEM.md REPLACES OpenCode's default provider prompt.** It does NOT append.

When Cortex sets `agent.cortex.prompt` in `OPENCODE_CONFIG_CONTENT`, OpenCode uses that prompt *instead of* its built-in provider-specific prompt (~106 lines of behavioral instructions). However, environment context and instruction files (AGENTS.md/CLAUDE.md) are still appended separately.

This contrasts with Claude Code ticket agents, where `--append-system-prompt` preserves Claude's built-in system prompt.

---

## How It Works

### Cortex Side (spawn flow)

1. **Resolve SYSTEM.md** — `internal/prompt/resolver.go` loads the appropriate SYSTEM.md (work/research/debug/chore)
2. **Generate config JSON** — `internal/core/spawn/opencode_config.go:32-65` embeds the system prompt into `agent.cortex.prompt`
3. **Pass via env var** — `spawn.go:346` sets `OPENCODE_CONFIG_CONTENT` with the JSON
4. **Launch OpenCode** — `launcher.go:168-186` runs `opencode --agent cortex --prompt "$(cat /tmp/prompt-file)"`

The resulting config:
```json
{
  "agent": {
    "cortex": {
      "description": "Cortex ticket agent",
      "mode": "bypassPermissions",
      "prompt": "<SYSTEM.md content here>",
      "permission": {"*": "allow"}
    }
  },
  "mcp": { ... }
}
```

### OpenCode Side (prompt assembly)

The critical code is in `session/llm.ts:66-79`:

```typescript
const system = []
system.push(
  [
    // use agent prompt otherwise provider prompt
    ...(input.agent.prompt
      ? [input.agent.prompt]         // <-- REPLACES provider prompt
      : isCodex ? [] : SystemPrompt.provider(input.model)),
    ...input.system,                 // <-- environment + instructions (APPENDED)
    ...(input.user.system ? [input.user.system] : []),
  ]
    .filter((x) => x)
    .join("\n"),
)
```

When `agent.prompt` is truthy (which it is for the "cortex" agent), `SystemPrompt.provider()` is **skipped entirely**.

### What IS Still Appended

These are passed via `input.system` from `prompt.ts:627`:

```typescript
system: [
  ...(await SystemPrompt.environment(model)),  // model name, cwd, git status, platform, date
  ...(await InstructionPrompt.system()),        // AGENTS.md, CLAUDE.md, custom instruction files
]
```

- **`SystemPrompt.environment()`** — model ID, working directory, git repo status, platform, date
- **`InstructionPrompt.system()`** — project AGENTS.md/CLAUDE.md, global AGENTS.md, `~/.claude/CLAUDE.md`, and any custom instruction files from config

---

## What Is Lost

The entire provider-specific prompt (`anthropic.txt`, ~106 lines) is replaced. Lost content includes:

| Category | Content Lost |
|----------|-------------|
| **Identity** | "You are OpenCode, the best coding agent on the planet" |
| **URL safety** | "You must NEVER generate or guess URLs" |
| **Help/feedback** | ctrl+p actions, GitHub issue link |
| **Self-help** | WebFetch from opencode.ai/docs guidance |
| **Tone/style** | No emojis, concise CLI output, markdown formatting, output text to communicate |
| **Professional objectivity** | Prioritize technical accuracy, disagree when necessary |
| **Task management** | TodoWrite tool usage patterns with examples |
| **Doing tasks** | General software engineering workflow |
| **Tool usage policy** | When to use Task vs Glob/Grep, parallel calls, specialized tools over bash |
| **Code references** | `file_path:line_number` formatting pattern |
| **File creation** | "NEVER create files unless absolutely necessary" |

### What Cortex's SYSTEM.md Actually Contains

For comparison, the Cortex `ticket/work/SYSTEM.md` is ~15 lines:

```markdown
# Role
You are a senior software engineer implementing features and fixes...

## Cortex Workflow
Use Cortex MCP tools: addComment, addBlocker, readReference, requestReview, concludeSession.
1. Understand the ticket requirements
2. Use readReference to read any referenced tickets/docs
3. Ask clarifying questions
4. Implement changes with appropriate tests
5. Verify your changes work
6. Call requestReview with a summary of changes
```

This covers **only** the Cortex workflow — no general coding behavior, tool usage patterns, or output formatting.

---

## Comparison: Claude Code vs OpenCode

| Aspect | Claude Code (ticket) | Claude Code (architect/meta) | OpenCode (all) |
|--------|---------------------|------------------------------|----------------|
| CLI flag | `--append-system-prompt` | `--system-prompt` | N/A (config JSON) |
| Behavior | **Appended** to built-in | **Replaces** built-in | **Replaces** provider prompt |
| Default prompt preserved? | Yes | No | No |
| General coding guidelines? | Yes (from Claude's built-in) | No (Cortex provides its own) | **No** |

There's an asymmetry: Claude Code ticket agents retain their built-in behavioral guidelines, while OpenCode agents lose them entirely.

---

## Recommendations

### Option 1: Prepend OpenCode's default prompt (Recommended)

In `GenerateOpenCodeConfigContent()`, prepend the provider prompt before the Cortex system prompt:

```go
func GenerateOpenCodeConfigContent(claudeConfig *ClaudeMCPConfig, systemPrompt string) (string, error) {
    // Combine: OpenCode default + Cortex system prompt
    combinedPrompt := openCodeDefaultPrompt + "\n\n" + systemPrompt
    config := OpenCodeConfigContent{
        Agent: map[string]OpenCodeAgentConfig{
            "cortex": {
                Prompt: combinedPrompt,
                // ...
            },
        },
    }
}
```

**Pros:** Restores all behavioral guidelines; matches Claude Code ticket agent behavior
**Cons:** Requires embedding/maintaining a copy of OpenCode's prompt; prompt grows larger; may drift from OpenCode upstream

### Option 2: Use instruction files instead of agent prompt

Move Cortex's system prompt into an instruction file (e.g., write it to `OPENCODE_CONFIG_DIR/AGENTS.md`) instead of using `agent.prompt`. This would cause it to be appended via `InstructionPrompt.system()`, preserving the default provider prompt.

**Pros:** Clean separation; no prompt duplication; auto-preserves future OpenCode prompt changes
**Cons:** Less control over prompt ordering; instructions appear as "Instructions from: /path" wrapper; may interact with other instruction files

### Option 3: Accept the loss

The model has training-time knowledge of coding best practices. The Cortex SYSTEM.md gives it a clear role and workflow. The lost guidelines (tool usage, formatting, etc.) may be largely redundant with what models already know.

**Pros:** Simplest; no changes needed; less prompt bloat
**Cons:** Loss of TodoWrite guidance, tool usage patterns, professional objectivity framing; may lead to suboptimal tool choices

### Option 4: Enrich Cortex's SYSTEM.md for OpenCode

Add the critical behavioral guidelines from OpenCode's default prompt into the Cortex SYSTEM.md templates, either as OpenCode-specific variants or as shared content.

**Pros:** Full control; can curate exactly what matters
**Cons:** Maintenance burden; diverges from OpenCode's upstream; needs per-agent-type prompt variants

---

## Key Source Files

| File | Purpose |
|------|---------|
| `cortex1/internal/core/spawn/opencode_config.go:32-65` | Generates OpenCode config with system prompt |
| `cortex1/internal/core/spawn/spawn.go:306-312` | Calls `GenerateOpenCodeConfigContent()` |
| `cortex1/internal/core/spawn/launcher.go:168-186` | Builds `opencode` CLI command |
| `opencode/packages/opencode/src/session/llm.ts:66-79` | System prompt assembly (the critical replace logic) |
| `opencode/packages/opencode/src/session/system.ts:19-27` | Provider-specific prompt selection |
| `opencode/packages/opencode/src/session/instruction.ts` | Instruction file loading (AGENTS.md, CLAUDE.md) |
| `opencode/packages/opencode/src/session/prompt/anthropic.txt` | Default Anthropic prompt (what's lost) |