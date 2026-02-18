---
id: 0e4c873a-ef24-4c5d-9492-29dc18a1c5ec
title: 'Research: OpenCode hooks for agent status updates (permissions, idle, etc.)'
type: research
tags:
    - research
    - opencode
    - agents
    - tmux
created: 2026-02-13T10:25:19.839039Z
updated: 2026-02-13T12:49:56.844718Z
---
## Goal

Investigate how Claude Code provides hooks/callbacks for agent status events (e.g., waiting for permissions, idle, working) and research whether OpenCode has equivalent mechanisms we can integrate with.

## Context

Claude Code has ways to detect when an agent is waiting for user permissions, which Cortex can hook into. We want similar integration for OpenCode so Cortex can be notified of agent state changes — especially permission prompts that block progress.

## Research Questions

1. How does Claude Code expose agent status events (waiting for permissions, tool approval, idle, etc.)?
2. What mechanism does Cortex currently use to hook into Claude Code's status?
3. Does OpenCode have any equivalent status reporting — CLI flags, output parsing, API, event hooks, or log signals?
4. If OpenCode doesn't have native support, what are the feasible approaches to detect agent state? (e.g., tmux pane output parsing, polling, process signals)
5. Are there any OpenCode CLI options, environment variables, or configuration that expose lifecycle events?

## Acceptance Criteria

- Document how Claude Code status hooks work in Cortex today
- Document what OpenCode offers (or doesn't) for equivalent functionality
- Propose feasible integration approaches for OpenCode status awareness
- Findings captured in a doc