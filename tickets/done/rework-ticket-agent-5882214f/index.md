---
id: 5882214f-452e-41e9-b9a2-2906f343f892
title: Rework ticket agent SYSTEM.md prompts
type: work
tags:
    - agents
    - cleanup
references:
    - ticket:2432d411-7c96-43f6-83db-bb6e61aad643
created: 2026-02-14T11:46:09.300082Z
updated: 2026-02-14T11:55:38.344526Z
---
## Problem

The SYSTEM.md prompts for ticket agents have several issues:

1. **Unconditional `readReference` instruction** — All SYSTEM.md prompts tell agents "Use `readReference` to read any referenced tickets or docs" as a first step, even when no references exist. This causes agents to hallucinate reference IDs and call `readReference` with invalid slugs, resulting in NOT_FOUND errors. The instruction should be conditional: only read references if they are listed in the kickoff.

2. **No error handling guidance** — None of the prompts tell agents what to do when stuck, when tests fail, or when the task is bigger than expected. No guidance on when to use `addBlocker`.

3. **Thin behavioral guidance** — The prompts list MCP tools and a numbered workflow but give little guidance on quality, communication style, or progress reporting expectations.

## Requirements

- Make the `readReference` step conditional — "if references are listed above, use `readReference` to read them" or similar phrasing
- Add guidance for what to do when blocked (use `addBlocker`, don't silently spin)
- Add guidance for when the task scope exceeds expectations (comment and flag, don't silently expand)
- Review and improve the workflow steps for each type (work, debug, research) to be more useful
- Keep prompts concise — don't over-instruct

## Note

The `chore` type is being removed in a parallel ticket (2432d411). Only rework SYSTEM.md for: **work**, **debug**, **research**.

## Acceptance Criteria

- No SYSTEM.md prompt unconditionally tells agents to read references
- All three SYSTEM.md prompts include error/blocker handling guidance
- Prompts remain concise and actionable
- Build passes