---
id: f0b6aaf8-3a82-48ce-b071-59a8faeb8e47
author: claude
type: done
created: 2026-02-14T11:49:37.437818Z
---
## Research Complete: OpenCode SYSTEM.md — Append vs Replace

### Key Finding
Cortex's SYSTEM.md **REPLACES** OpenCode's default provider prompt when injected via `agent.cortex.prompt` in `OPENCODE_CONFIG_CONTENT`. OpenCode's `llm.ts:71` skips its built-in ~106-line behavioral prompt entirely when an agent has a custom prompt set.

### What's Preserved
- Environment context (model name, cwd, git status, platform, date) — appended separately via `SystemPrompt.environment()`
- Instruction files (AGENTS.md, CLAUDE.md) — loaded independently via `InstructionPrompt.system()`

### What's Lost
OpenCode's entire provider-specific prompt (`anthropic.txt`), including: tool usage policies, TodoWrite guidance, professional objectivity framing, tone/style rules, parallel tool call instructions, file creation guidelines, and code reference formatting.

### Asymmetry with Claude Code
- Claude Code ticket agents: `--append-system-prompt` → default prompt preserved
- Claude Code architect/meta: `--system-prompt` → default prompt replaced (intentional)
- OpenCode (all agent types): `agent.prompt` → default prompt replaced

### Deliverable
Created findings doc "OpenCode System Prompt: Append vs Replace Analysis" with full trace through both codebases, comparison table, and 4 remediation options. Recommended Option 2: use instruction files (`AGENTS.md` in config dir) instead of `agent.prompt` to preserve OpenCode's default prompt.

### Files Analyzed (no changes made)
- `cortex1/internal/core/spawn/opencode_config.go` — config generation
- `cortex1/internal/core/spawn/spawn.go` — spawn orchestration
- `cortex1/internal/core/spawn/launcher.go` — CLI command builder
- `opencode/packages/opencode/src/session/llm.ts` — system prompt assembly (critical replace logic)
- `opencode/packages/opencode/src/session/system.ts` — provider prompt selection
- `opencode/packages/opencode/src/session/instruction.ts` — instruction file loading
- `opencode/packages/opencode/src/session/prompt/anthropic.txt` — default prompt content