---
id: 1b2c5df9-7af3-464b-88d2-dcee729417e6
title: 'Simplify cortex meta: always fresh, remove concludeSession'
type: work
tags:
    - cleanup
    - cli
    - mcp
    - meta-agent
created: 2026-02-14T12:50:36.866312Z
updated: 2026-02-14T13:00:53.591947Z
---
## Problem

`cortex meta` has unnecessary complexity:

1. **`--mode` flag** (normal/resume/fresh) — meta sessions should always start fresh. There's no value in resuming a meta session.
2. **`concludeSession` MCP tool** — meta doesn't need session summaries. Users start meta, do their work, and close it. The conclude ceremony is useless overhead.

## Requirements

### Always fresh
- Remove the `--mode` flag from `cortex meta`
- Meta sessions always spawn in fresh mode
- If an orphaned meta session exists, silently clean it up and start fresh

### Remove concludeSession from meta
- Remove `concludeSession` from the meta MCP tool set (`tools_meta.go`)
- Remove any session summary doc creation logic specific to meta sessions
- Meta session cleanup should happen silently when a new meta session starts (or when the tmux window closes)

### Keep `--detach`
- The `--detach` flag is still useful — keep it

## Acceptance Criteria

- `cortex meta` always starts a fresh session with no mode prompt
- No `concludeSession` tool available in meta MCP sessions
- Orphaned meta sessions are cleaned up automatically
- `--detach` still works
- Build passes, tests pass