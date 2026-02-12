---
id: 6a11cbff-6e59-47de-801d-543a56a76b93
author: claude
type: done
created: 2026-02-11T08:44:03.864884Z
---
## Summary

Investigated OpenCode's system prompt injection capabilities by tracing the full prompt assembly pipeline through the source at ~/ephemeral/opencode. Confirmed there is no `--system-prompt` CLI flag and no `SYSTEM.md` file convention.

### Key Findings

1. **Prompt assembly pipeline** (`llm.ts` → `system.ts` → `instruction.ts` → `config.ts`): The system prompt is built in layers — agent/provider prompt, environment info, instruction files, and plugin transforms.

2. **Replace vs. Append distinction**: Agent prompts (defined via markdown files, JSON config, or `OPENCODE_CONFIG_CONTENT`) **replace** the provider-specific base prompt. Instruction files and context files (AGENTS.md, CLAUDE.md) are always **appended**. Environment info (model ID, cwd, platform, date) is always injected and cannot be suppressed except via plugin hook.

3. **6 injection methods identified**:
   - Agent markdown file (`.opencode/agents/*.md`) — recommended for static configs
   - JSON config `prompt` field with `{file:path}`/`{env:VAR}` substitution
   - `OPENCODE_CONFIG_CONTENT` env var — highest-priority but no variable substitution (literal JSON only)
   - `instructions` config field — appends supplementary files/URLs/globs
   - Context files (AGENTS.md, CLAUDE.md, CONTEXT.md) — auto-loaded, appended
   - Plugin `experimental.chat.system.transform` hook — full programmatic control

4. **`OPENCODE_CONFIG_CONTENT` caveat**: Uses `JSON.parse()` directly, bypassing the `load()` function — `{file:}` and `{env:}` substitution do NOT work.

### Deliverables

- Created research doc "OpenCode System Prompt Injection: Complete Guide" (id: cb80accd) with full comparison matrix, source-level code references, and integration recommendations.
- Updates previous research doc (id: b111544a) with deeper technical detail on the prompt pipeline.

### Recommendations for Next Steps

1. **For Cortex OpenCode integration**: Use `OPENCODE_CONFIG_CONTENT` with an inline agent definition + `--agent` flag for dynamic per-ticket prompt injection. The command pattern would be:
   ```bash
   OPENCODE_CONFIG_CONTENT='{"agent":{"cortex":{"mode":"primary","prompt":"<system prompt>"}}}' opencode run --agent cortex "<message>"
   ```

2. **For static setups**: Create `.opencode/agents/cortex-ticket.md` with the Cortex ticket agent prompt as the markdown body and appropriate frontmatter (permissions, model, etc.).

3. **For maximum control**: Consider writing a small OpenCode plugin using the `experimental.chat.system.transform` hook to dynamically inject Cortex-specific context (ticket details, project info) into the system prompt at runtime.