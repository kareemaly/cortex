---
id: d7ef8c16-58a9-4854-80c1-8e97d54857ac
title: Research OpenCode CLI integration with Cortex
type: research
tags:
    - research
    - opencode
created: 2026-02-11T08:08:01.791808Z
updated: 2026-02-11T08:27:11.240798Z
---
## Objective

Investigate OpenCode CLI and determine how it can be integrated into Cortex as a supported agent type. OpenCode is already referenced in the codebase config (`opencode` agent type) — this research should determine what's already wired up vs what's missing, and how OpenCode compares to Claude Code and Copilot in terms of integration capabilities.

## Research Steps

1. **Understand OpenCode CLI** — Run `opencode --help` and explore its subcommands, flags, and configuration options. Understand how it operates (interactive vs headless, session management, input/output model, customizability).

2. **Check existing Cortex support** — The codebase already references `opencode` as an agent type. Investigate what's already implemented and what gaps remain.

3. **Identify integration points** — Determine what OpenCode needs to work as a full Cortex ticket agent:
   - How to launch it in a tmux pane (headless/non-interactive mode?)
   - How to pass initial instructions (KICKOFF prompt equivalent)
   - Whether it supports MCP tool injection
   - How to pass environment variables or config
   - Session lifecycle (does it exit on completion? can it be resumed?)
   - Customizability — how configurable is it compared to other agents?

4. **Document gaps and blockers** — Note any capabilities OpenCode lacks that would prevent full integration.

## Deliverable

Create a doc summarizing findings, with a clear recommendation on how to complete or improve OpenCode as a supported agent type in Cortex.