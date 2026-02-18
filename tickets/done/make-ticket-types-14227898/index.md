---
id: 14227898-240e-4cef-a07e-0e1706b8b7dc
title: Make ticket types fully config-driven
type: work
tags:
    - agents
    - mcp
    - tui
    - configuration
created: 2026-02-17T15:41:55.927676Z
updated: 2026-02-17T16:18:49.291671Z
---
## Problem

Ticket types are partially hardcoded in the codebase. While the config-driven validation already works (types come from the `ticket` map in `cortex.yaml`), several places have hardcoded assumptions about `debug` and `research` types — breaking flexibility for users who want to define their own types like `frontend`, `infra`, `spike`, etc.

## Requirements

### 1. Give `createDoc` tool to all ticket workers
- Remove the `if type == "research"` guard in MCP ticket tool registration
- All ticket agents get `createDoc` regardless of type

### 2. Prompt fallback chain
- When resolving prompts for a ticket type, use this fallback:
  - `prompts/ticket/{type}/{stage}.md` → `prompts/ticket/work/{stage}.md` → embedded defaults
- This means a user can define a `frontend` type without providing custom prompts — it falls back to `work` prompts

### 3. Remove debug/research from embedded defaults
- Delete `internal/install/defaults/main/prompts/ticket/debug/` directory
- Delete `internal/install/defaults/main/prompts/ticket/research/` directory
- Keep only `internal/install/defaults/main/prompts/ticket/work/` as the universal default

### 4. Generic kanban badge colors
- Kanban badge rendering currently hardcodes colors for `debug` (red) and `research` (blue)
- Replace with a generic approach that works for any type name (e.g., hash-based color assignment, or a small rotating palette)
- `work` type should continue to show no badge (it's the default)

### 5. Remove all hardcoded type references
- Eliminate every reference to `debug` and `research` as ticket types from the source code
- This includes: MCP tool descriptions, CLAUDE.md documentation, meta/architect agent system prompts, API handler comments, type constants, etc.
- After this work, the only place `debug` or `research` should appear is in a project's own `.cortex/cortex.yaml` as user-configured types

### 6. Update documentation
- CLAUDE.md: update ticket type documentation to reflect they are user-defined
- Meta agent system prompt: remove references to canonical three types
- MCP tool descriptions: update `createTicket` description to not enumerate fixed types

## Acceptance Criteria
- A user can define any ticket type name in `cortex.yaml` and it works end-to-end (create → spawn → prompts resolve → kanban renders badge)
- All ticket agents get `createDoc` tool
- Types without custom prompts gracefully fall back to `work` prompts
- Kanban renders a colored badge for any non-`work` type
- Zero references to `debug` or `research` as ticket type values in Go source, embedded defaults, or documentation (only in project-level `.cortex/cortex.yaml`)
- `cortex init` still scaffolds default types in the generated config (these are just suggestions, not hardcoded elsewhere)