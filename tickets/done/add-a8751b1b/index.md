---
id: a8751b1b-f56f-44a1-b17d-384577decc89
title: Add addTicketComment to Architect MCP Tools
type: ""
created: 2026-01-28T07:24:07.327263Z
updated: 2026-01-28T07:32:47.394528Z
---
## Summary

Expose `addTicketComment` in the architect MCP tool set so the architect can add comments to any ticket, not just read them.

## Motivation

The architect needs to:
- Respond to agent questions (blocker comments)
- Add architectural guidance mid-flight on tickets in progress
- Record decisions as comments (type: decision)
- Provide review feedback on completed work

Currently `addTicketComment` is only available in the ticket agent MCP session. The architect can only communicate through ticket body updates, which is insufficient.

## Changes

### 1. Update `internal/daemon/mcp/tools_architect.go`

Add `addTicketComment` to the architect tool registration. Unlike the ticket agent version (which implicitly uses the session's ticket ID), the architect version should require a ticket ID parameter since the architect operates across all tickets.

## Acceptance Criteria

- Architect MCP session exposes `addTicketComment` tool
- Tool requires a ticket ID parameter (architect is not scoped to a single ticket)
- Tool supports all comment types: decision, blocker, progress, question, scope_change, general
- Existing ticket agent `addTicketComment` behavior is unchanged