---
id: b31c011c-86ca-4234-87f8-baa9cef2a638
title: Remove meta agent entirely
type: work
tags:
    - meta-agent
    - cleanup
    - mcp
    - api
created: 2026-02-17T19:50:29.502698Z
updated: 2026-02-17T20:57:28.477056Z
---
## Problem

The meta agent (top-level in the Meta → Architect → Ticket Agent hierarchy) is unused and adds unnecessary complexity. Remove it entirely — no references to meta should remain in the codebase after this work.

## What to Remove

### CLI
- `cortex meta` command

### MCP
- `tools_meta.go` — the entire meta MCP tool set (`listProjects`, `registerProject`, `unregisterProject`, `spawnArchitect`, `listSessions`, `readProjectConfig`, `updateProjectConfig`, `readGlobalConfig`, `updateGlobalConfig`, `readPrompt`, `updatePrompt`, `readDaemonLogs`, `daemonStatus`)
- Meta session type / mode in MCP server setup

### API
- `/meta/*` endpoints
- `MetaSessionManager` and its global session file (`~/.cortex/meta-session.json`)

### Session Management
- Meta session state detection, orphan recovery, spawn logic
- Any meta-specific session tracking in `.cortex/sessions.json` or global state

### Default Prompts
- `prompts/meta/SYSTEM.md` from embedded defaults

### Documentation
- All references in CLAUDE.md (three-tier hierarchy, meta commands, meta MCP tools table, meta endpoints, meta session description)
- Agent system prompts that reference meta (architect SYSTEM.md, etc.)

## Important
- **Do not remove tools that meta happens to expose if they're also useful elsewhere.** For example, if `listProjects` is also available to architects, keep it there. The goal is removing the meta *agent layer*, not necessarily every tool it had — check what architects already have access to.
- Clean up any dead imports, unused types, and test code related to meta.

## Acceptance Criteria
- `cortex meta` command no longer exists
- No meta MCP tools file or meta session type
- No meta API endpoints or MetaSessionManager
- No meta prompts in embedded defaults
- Zero references to "meta" as an agent tier/session type in Go source or documentation
- The codebase compiles and existing tests pass