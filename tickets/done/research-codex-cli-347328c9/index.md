---
id: 347328c9-ac21-4075-bc22-f3f5f04d09a9
title: Research Codex CLI integration with Cortex
type: research
tags:
    - research
    - codex
created: 2026-02-11T07:53:25.021661Z
updated: 2026-02-11T08:06:16.823434Z
---
## Objective

Investigate OpenAI's Codex CLI tool and determine how it can be integrated into Cortex as a supported agent type (alongside the existing `claude` and `copilot` agents).

## Research Steps

1. **Understand Codex CLI** — Run `codex --help` and explore its subcommands, flags, and configuration options. Understand how it operates (interactive vs headless, session management, input/output model).

2. **Compare with existing agent integrations** — Look at how Cortex currently integrates with Claude Code and Copilot. Understand the agent spawn lifecycle, how MCP tools are injected, how prompts/instructions are passed, and how sessions are managed in tmux.

3. **Identify integration points** — Determine what Codex needs to work as a Cortex ticket agent:
   - How to launch it in a tmux pane (headless/non-interactive mode?)
   - How to pass initial instructions (KICKOFF prompt equivalent)
   - Whether it supports MCP tool injection
   - How to pass environment variables or config
   - Session lifecycle (does it exit on completion? can it be resumed?)

4. **Document gaps and blockers** — Note any capabilities Codex lacks that would prevent full integration (e.g., no MCP support, no way to inject tools, no headless mode).

## Deliverable

Create a doc summarizing findings, with a clear recommendation on whether and how to add Codex as a supported agent type in Cortex.