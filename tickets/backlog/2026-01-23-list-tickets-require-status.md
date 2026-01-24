# List Tickets Require Status

## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

The `listTickets` MCP tool has status as optional. Architect should explicitly choose which status to query.

## Requirements

- Make `status` parameter required in `listTickets` MCP tool
