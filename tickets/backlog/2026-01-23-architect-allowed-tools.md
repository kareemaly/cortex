# Architect Allowed Tools

## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

Architect spawns with `--permission-mode plan` which is too restrictive. Should use allowed tools instead.

## Requirements

- Remove `--permission-mode plan` from architect spawn command
- Add `--allowedTools "mcp__cortex__createTicket,mcp__cortex__readTicket"` to architect spawn command
- Keep `--permission-mode plan` for ticket agents (unchanged)
