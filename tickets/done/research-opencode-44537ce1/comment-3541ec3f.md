---
id: 3541ec3f-d58b-42c5-be7a-bc032697efda
author: claude
type: review_requested
created: 2026-02-11T08:40:44.289639Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Research Complete: OpenCode System Prompt Injection Methods

### Key Finding
OpenCode has **no `--system-prompt` CLI flag** and **no `SYSTEM.md` convention** — the previous research was correct on this point. However, there are **6 distinct injection methods** available, which I've documented with source-level tracing.

### What Was Investigated
- Traced the full prompt assembly pipeline: `llm.ts` → `system.ts` → `instruction.ts` → `config.ts`
- Verified all CLI flags (none for system prompt)
- Verified all env vars (`OPENCODE_CONFIG_CONTENT` is the closest, but has caveats)
- Mapped the agent loading pipeline from markdown files and JSON config
- Checked variable substitution (`{file:}`, `{env:}`) scope

### Critical Insight: Replace vs. Append
- **Agent prompt** (Methods 1-3) **replaces** the provider-specific base prompt (Anthropic/Gemini/etc.)
- **Instruction files** (Methods 4-5) are always **appended** — they don't replace anything
- **Environment info** (model ID, cwd, platform, date) is **always** injected — cannot be suppressed except via plugin hook
- **Plugin transform** (Method 6) has full control over the entire system array

### 6 Injection Methods (ranked by Cortex relevance)
1. **Agent markdown file** (`.opencode/agents/*.md`) — body replaces provider prompt
2. **JSON config `prompt` field** — supports `{file:path}` and `{env:VAR}` substitution
3. **`OPENCODE_CONFIG_CONTENT` env var** — highest-priority config, but NO `{file:}`/`{env:}` substitution (uses `JSON.parse()` directly)
4. **`instructions` config field** — appends files/URLs/globs to system prompt
5. **Context files** (`AGENTS.md`, `CLAUDE.md`, `CONTEXT.md`) — auto-loaded and appended
6. **Plugin `experimental.chat.system.transform` hook** — full programmatic control

### Cortex Integration Recommendation
For spawning OpenCode as a ticket agent, **Method C** (`OPENCODE_CONFIG_CONTENT` with inline agent definition + `--agent` flag) is best for dynamic per-ticket prompt injection. For static setups, **Method A** (agent markdown file) is simplest.

### Deliverable
Created research doc "OpenCode System Prompt Injection: Complete Guide" with full comparison matrix, code references, and integration recommendations.