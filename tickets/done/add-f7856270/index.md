---
id: f7856270-6c3e-4933-98ea-eb7bcf49aa50
title: Add `getCortexConfigDocs` MCP tool for architect
type: work
created: 2026-02-03T08:45:37.415997Z
updated: 2026-02-03T08:53:50.17467Z
---
# Overview

Add an MCP tool available to architects that returns the embedded config documentation for the project's agent type.

## Tool Definition

**Name**: `getCortexConfigDocs`

**Parameters**: None

**Returns**: Contents of `CONFIG_DOCS.md` for the project's configured agent type

## Implementation

1. Add tool to architect MCP tools in `internal/daemon/mcp/tools_architect.go`
2. Read agent type from project config (`architect.agent`, default: `claude`)
3. Return contents of embedded `internal/install/defaults/<agent-type>/CONFIG_DOCS.md`

## Behavior

```
Architect calls: getCortexConfigDocs()

Returns: Full markdown content of ~/.cortex/defaults/claude-code/CONFIG_DOCS.md
(or embedded equivalent)
```

## Notes

- Only available to architect sessions (not ticket agents)
- Returns docs for the agent type configured in the project
- Currently only `claude-code` exists, but structure supports future agent types