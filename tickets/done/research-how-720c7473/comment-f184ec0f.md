---
id: f184ec0f-eb55-432b-889d-1ce94744ff49
author: claude
type: review_requested
created: 2026-02-14T11:46:47.155137Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Research Complete: OpenCode System Prompt — Append vs Replace

### Key Finding

**Cortex's SYSTEM.md REPLACES OpenCode's default provider prompt.** When the `agent.cortex.prompt` field is set in `OPENCODE_CONFIG_CONTENT`, OpenCode skips its built-in ~106-line behavioral prompt entirely (`session/llm.ts:71`).

### What's preserved
- Environment context (model name, cwd, git status, platform, date) — appended separately
- Instruction files (AGENTS.md, CLAUDE.md) — loaded independently via `InstructionPrompt.system()`

### What's lost
OpenCode's entire provider prompt including: tool usage policies, TodoWrite guidance, professional objectivity framing, tone/style rules, parallel tool call instructions, file creation guidelines, and code reference formatting.

### Asymmetry with Claude Code
- **Claude Code ticket agents**: `--append-system-prompt` → default prompt preserved
- **Claude Code architect/meta**: `--system-prompt` → default prompt replaced (intentional)
- **OpenCode (all agent types)**: `agent.prompt` → default prompt replaced

### Recommended approach
**Option 2: Use instruction files** — Move Cortex's system prompt into `OPENCODE_CONFIG_DIR/AGENTS.md` instead of `agent.prompt`. This preserves OpenCode's default provider prompt and auto-tracks upstream changes. Clean separation, no prompt duplication.

Full analysis with 4 options documented in findings doc.