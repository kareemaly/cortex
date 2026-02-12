---
id: 245b6ad5-4f48-4106-a270-2dd971197e73
title: Clarify that concludeSession auto-creates session doc
type: chore
tags:
    - architect
    - cleanup
    - session
created: 2026-02-10T10:19:01.275865Z
updated: 2026-02-10T10:22:16.669303Z
---
## Problem

The architect sometimes manually creates a session summary doc before calling `concludeSession`, not realizing that `concludeSession` already creates one automatically. This leads to duplicate session docs.

## Changes Needed

1. **Architect SYSTEM.md** — add clear instruction that `concludeSession` automatically persists a session summary doc, and the architect should NOT create one manually before concluding
2. **`concludeSession` MCP tool description** — update the description text to explicitly state it creates a session doc as part of cleanup

## Acceptance Criteria

- Architect SYSTEM.md clearly states concludeSession handles session doc creation
- `concludeSession` tool description mentions session doc creation
- No behavioral ambiguity about who creates the session doc