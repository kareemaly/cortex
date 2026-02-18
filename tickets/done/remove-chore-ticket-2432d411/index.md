---
id: 2432d411-7c96-43f6-83db-bb6e61aad643
title: Remove chore ticket type entirely
type: work
tags:
    - agents
    - cleanup
created: 2026-02-14T11:43:21.198874Z
updated: 2026-02-14T11:55:16.475965Z
---
## Problem

The `chore` ticket type is nearly identical to `work` — same kickoff template, near-identical APPROVE, and the SYSTEM prompt difference is trivial. It doesn't justify a separate type.

## Requirements

- Remove `chore` as a valid ticket type everywhere in the codebase
- Delete the chore prompt templates (`prompts/ticket/chore/SYSTEM.md`, `KICKOFF.md`, `APPROVE.md`)
- Update the ticket type validation to only accept: `work`, `debug`, `research`
- Update any MCP tool definitions, API handlers, or CLI commands that reference `chore` as a valid type
- Update any documentation or prompt text that lists ticket types (e.g., architect SYSTEM.md mentions "chore" in the ticket types list)
- Migrate any existing `chore` tickets in the repo to type `work`

## Breaking Change

This is intentional. No backward compatibility needed — no external users.

## Acceptance Criteria

- `chore` is no longer accepted as a ticket type anywhere
- Creating a ticket with `type: chore` returns an error
- Chore prompt directory is deleted from defaults
- All existing chore tickets (if any) are migrated to `work`
- Build passes, tests pass