---
id: 383f93da-d530-470c-b4e6-c92eae1eb80f
title: Fix ticket view flashing on agent status updates
type: work
tags:
    - tui
created: 2026-02-18T07:38:19.603857Z
updated: 2026-02-18T07:47:10.236521Z
---
## Problem

The ticket detail TUI view flashes constantly when an agent is actively working on a ticket. Every SSE event (agent status updates) triggers a full loading state, replacing the view content with "Loading ticket..." for ~100-300ms before re-rendering with fetched data. This creates a distracting flash on every status update.

## Root Cause

In the ticket view's `EventMsg` handler, `m.loading = true` is set on every SSE event. The `View()` method has an early exit that shows "Loading ticket..." when `m.loading` is true. This creates a visible content replacement on every background refresh.

The dashboard view does NOT have this problem because it loads data asynchronously without setting a loading flag — old content stays visible until new data arrives.

## Requirements

- SSE-triggered refreshes should NOT show a loading state
- Keep displaying current ticket content while new data loads in the background
- Silently swap in updated data when it arrives (like the dashboard does)
- Only show the "Loading ticket..." state on initial ticket load (when there's no existing content to display)
- Preserve viewport scroll position across background refreshes

## Acceptance Criteria

- No visible flashing when an agent is actively working on a ticket
- Ticket view still updates with new data (comments, status changes) as they happen
- Initial load still shows a loading indicator
- Scroll position is preserved across updates