---
id: cfbf11a8-40ca-47eb-9093-3e73d7055e3c
author: claude
type: review_requested
created: 2026-02-11T08:01:50.521738Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Research Complete: Codex CLI Integration with Cortex

### Summary

Investigated OpenAI's Codex CLI (v0.77.0) for integration as a third supported agent type in Cortex alongside `claude` and `copilot`.

**Verdict: Integration is feasible and recommended.** Codex follows the Copilot pattern — no system prompt flag, no hooks support, instructions embedded in KICKOFF.md.

### Key Findings

1. **Codex has full MCP support** — can register external MCP servers via `codex mcp add` with stdio transport, environment variables, and command args. This means the existing cortexd MCP server works as-is.

2. **The main challenge is MCP config injection.** Claude and Copilot accept `--mcp-config <file>` flags for per-session MCP config. Codex uses `codex mcp add/remove` which persists to a global `~/.codex/config.toml`. The recommended approach is wrapping registration/cleanup in the launcher script.

3. **Concurrent Codex agents are limited** — because MCP config is global, only one Codex agent can use the "cortex" MCP server name at a time. This is the most significant constraint. Potential mitigations (config profiles, `-c` overrides) need testing.

4. **Implementation follows the Copilot template** — add `AgentCodex` type, `buildCodexCommand()` launcher, skip settings/hooks, create default prompts. Roughly ~15 files modified/created.

5. **Five open questions** remain that require hands-on testing before implementation: MCP tool name prefix format, config profile MCP scoping, runtime `-c` overrides, project-level config directories, and interactive vs exec mode for tmux.

### Deliverable

Created research doc: "Codex CLI Integration with Cortex — Research Findings" with:
- Full capability comparison table (Codex vs Claude vs Copilot)
- Four MCP injection approaches analyzed with trade-offs
- Complete implementation plan (files to modify/create)
- Proposed default config and launcher script structure
- Gaps, blockers, and risks documented
- Open questions for pre-implementation testing