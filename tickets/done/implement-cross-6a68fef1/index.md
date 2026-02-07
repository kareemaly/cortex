---
id: 6a68fef1-0b9b-42fb-8d29-f7d264ef754a
title: Implement cross-project ticketing for architects
type: work
created: 2026-02-05T09:35:19.060173Z
updated: 2026-02-05T09:46:43.034974Z
---
## Summary

Enable architects to access, create, and spawn tickets across different projects from a single MCP session.

## Context

Research ticket `45b4ec7d` confirmed this is feasible. The daemon infrastructure (StoreManager, HTTP API) already supports multi-project access — the constraint is only at the MCP layer where SDK client binds to one project at init.

## Requirements

### 1. Add `listProjects` architect tool
- Exposes existing `GET /projects` endpoint to MCP
- Returns path, title, and ticket counts for each registered project
- No parameters required

### 2. Add `WithProject()` method to SDK client
- Per-request override of the `X-Cortex-Project` header
- Located in `internal/cli/sdk/client.go`

### 3. Add optional `project_path` parameter to architect tools
- If omitted → uses session default (backward compatible)
- If provided → validates against registry, uses specified project
- Apply to all tools except `deleteTicket`

### 4. Validation
- Target project must be registered in `~/.cortex/settings.yaml`
- Return clear error if project not found in registry

## Implementation Notes

**Files to modify:**
- `internal/cli/sdk/client.go` — Add `WithProject(path)` method
- `internal/daemon/mcp/tools_architect.go` — Add `listProjects` tool, add `project_path` param to schemas

**Project identification:**
- Use absolute paths only (not names/titles)
- Names can collide, paths cannot

## Out of Scope
- CLI cross-project support (already works via `--project` flag or changing directories)
- Cross-project `deleteTicket` (too risky)