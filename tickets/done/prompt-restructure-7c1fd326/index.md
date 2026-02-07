---
id: 7c1fd326-46fc-4a0a-83b8-f35825251bd4
title: Prompt Restructure
type: ""
created: 2026-01-24T10:37:13Z
updated: 2026-01-24T10:37:13Z
---
## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

Current prompts mix dynamic content with static instructions. Need cleaner separation using `--append-system-prompt`.

## Requirements

### Ticket Agent

**Prompt (hardcoded in code):**
```
# Ticket: {{.Title}}

{{.Body}}
```

**System prompt:** `--append-system-prompt .cortex/prompts/ticket-agent.md`

### Architect

**Prompt (hardcoded in code, queries daemon API at spawn):**
```
# Tickets

## Backlog
- [title] (updated: 2026-01-24)
...

## In Progress
...

## Review
...

## Done (recent 5)
...
```

**System prompt:** `--append-system-prompt .cortex/prompts/architect.md`

### Prompt Files

Must be installed via `cortex init`. Error if missing.

Update tool names to fully qualified format:
- `mcp__cortex__createTicket`
- `mcp__cortex__readTicket`
- `mcp__cortex__listTickets`
- `mcp__cortex__updateTicket`
- `mcp__cortex__deleteTicket`
- `mcp__cortex__moveTicket`
- `mcp__cortex__spawnSession`
- `mcp__cortex__pickupTicket`
- `mcp__cortex__submitReport`
- `mcp__cortex__approve`

### Cleanup

- Remove template rendering from prompt loading
- Remove default prompts from code

## Implementation

### Commits

- `624d4f7` feat: separate dynamic content from static prompt instructions via --append-system-prompt
- `ba5fae9` fix: pass file content to --append-system-prompt instead of path

### Key Files Changed

- `internal/core/spawn/command.go` - Added `AppendSystemPrompt` field (now passes content not path), escapes content for shell
- `internal/core/spawn/spawn.go` - New `buildPrompt()` logic that queries daemon API for architect, builds dynamic content, loads prompt file content
- `internal/prompt/prompt.go` - Removed template rendering, kept path helpers, added `ValidatePromptFile()` and `LoadPromptFile()`
- `internal/prompt/errors.go` - Removed `ParseError` and `RenderError`, kept `NotFoundError`
- `internal/install/install.go` - Added default prompts as local constants with qualified MCP tool names

### Key Decisions

- Moved slug generation inline in `command.go` to avoid circular dependency with ticket package
- Architect prompt includes project name header followed by ticket list grouped by status
- Ticket agent prompt is minimal: just title and body (static instructions come from file)
- Default prompts use fully qualified MCP tool names (e.g., `mcp__cortex__listTickets`)
- `--append-system-prompt` receives file content (not path) - content is escaped for shell using POSIX single-quote escaping

### Scope Changes

- Added project name to architect dynamic prompt (not in original spec but requested during implementation)