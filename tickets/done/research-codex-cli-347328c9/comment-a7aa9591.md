---
id: a7aa9591-cae8-4dfc-bfef-a8d5ff2087f2
author: claude
type: done
created: 2026-02-11T08:06:16.82196Z
---
## Summary

Researched OpenAI's Codex CLI (v0.77.0) for integration as a supported agent type in Cortex alongside `claude` and `copilot`.

### Key Findings

1. **Integration is feasible and recommended.** Codex has interactive TUI mode, full MCP support (stdio transport), session resume, sandbox isolation, and configurable approval policies — all needed for Cortex's agent lifecycle.

2. **Follows the Copilot pattern.** No `--system-prompt` flag, no `--settings` hooks. All workflow instructions go into KICKOFF.md templates. Implementation mirrors the existing Copilot integration.

3. **MCP injection is the core challenge.** Codex uses `codex mcp add/remove` (global config) instead of a `--mcp-config <file>` flag. Recommended approach: wrap MCP registration/cleanup in the launcher script. This limits concurrent Codex agents to one at a time.

4. **Implementation scope:** ~15 files to modify/create — add `AgentCodex` config type, `buildCodexCommand()` launcher, default config + prompt templates, skip settings generation.

### Deliverables

- Created research doc: "Codex CLI Integration with Cortex — Research Findings" covering full capability analysis, 4 MCP injection approaches with trade-offs, implementation plan, proposed default configs, gaps/blockers, and open questions.

### Recommended Next Steps

1. **Hands-on testing** to answer the 5 open questions (MCP tool prefix format, config profiles for MCP scoping, `-c` runtime overrides, project-level config, interactive vs exec mode in tmux).
2. **Create implementation ticket** using the plan from the research doc — config changes, launcher builder, default prompts, tests.
3. **Consider requesting `--mcp-config <file>` from OpenAI** as a feature request to align with the Claude/Copilot pattern and unblock concurrent sessions.