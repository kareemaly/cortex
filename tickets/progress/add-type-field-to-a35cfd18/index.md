---
id: a35cfd18-9878-4121-afe0-63959b2b2745
title: Add type field to updateTicket API and MCP tool
type: work
tags:
    - api
    - mcp
created: 2026-02-18T08:02:25.35896Z
updated: 2026-02-18T08:05:13.947958Z
---
## Problem

The `updateTicket` API endpoint and MCP tool accept title, body, tags, and references — but not `type`. Ticket type can only be set at creation time via `createTicket`. There's no way to change a ticket's type after creation.

## Requirements

- Add `type` as an optional field to the update ticket API endpoint
- Add `type` to the `updateTicket` MCP tool input schema
- Validate the type against project config (same validation as `createTicket`)
- If type is provided, update the ticket's type in the store

## Acceptance Criteria

- `updateTicket` MCP tool accepts an optional `type` parameter
- HTTP PATCH/PUT endpoint for tickets accepts `type` in the request body
- Invalid types are rejected with an error listing valid types (matching `createTicket` behavior)
- Omitting `type` leaves the existing type unchanged