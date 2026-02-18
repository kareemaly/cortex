---
id: 720c7473-2f6f-4296-b7fd-e4d4bf188cdb
title: 'Research: How OpenCode handles SYSTEM.md — append vs replace'
type: research
tags:
    - research
    - opencode
    - agents
created: 2026-02-14T11:41:29.153519Z
updated: 2026-02-14T11:49:37.439265Z
---
## Question

When Cortex spawns a ticket agent session with OpenCode, does the SYSTEM.md prompt get **appended** to OpenCode's existing system prompt or does it **replace** it entirely?

For Claude Code, we know SYSTEM.md is appended via `--append-system-prompt`. We need to understand OpenCode's equivalent behavior.

## Investigation Scope

- Explore `~/ephemeral/opencode` to understand how OpenCode handles system prompts
- Look at how OpenCode accepts custom system prompt content (CLI flags, config, environment variables)
- Determine whether OpenCode has a concept of "append to system prompt" vs "replace system prompt"
- Check how Cortex currently passes SYSTEM.md to OpenCode during spawn (look at spawn orchestration logic)
- Document what OpenCode's default system prompt contains and whether we'd lose important behavior by replacing it

## Acceptance Criteria

- Clear answer: does the current Cortex spawn for OpenCode append or replace the system prompt?
- Document OpenCode's system prompt injection mechanism
- If it replaces, document what default OpenCode system prompt content is lost
- Recommendation on whether we need to change the approach