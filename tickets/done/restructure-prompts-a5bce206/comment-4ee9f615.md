---
id: 4ee9f615-98d2-47ff-89ba-dd8f67dc6bd8
author: claude
type: ticket_done
created: 2026-01-28T09:37:14.834088Z
---
## Summary

Restructured the cortex project's config schema, ticket model, and prompt system from a flat layout to a type-based architecture supporting multiple ticket types (e.g., "work", "investigation"). This was a 6-phase implementation touching 25 files across 10 packages.

## Changes Made

### Phase 1: Config Schema
- Replaced `AgentArgsConfig` struct with `RoleConfig` (Agent + Args) and `TicketConfig` (map[string]RoleConfig)
- Config now uses nested `architect`/`ticket` structure with per-type agent and args
- Added `TicketRoleConfig(ticketType)` helper method
- Updated `DefaultConfig()` and `Validate()`

### Phase 2: Ticket Model
- Added `DefaultTicketType = "work"` constant and `Type string` field to `Ticket`
- Changed `store.Create(title, body)` → `store.Create(title, body, ticketType)` with empty defaulting to "work"
- Propagated `Type` field through response types, API types, MCP types, SDK client, and all conversion functions

### Phase 3: Prompt System
- Replaced 6 flat path functions with 2 type-based functions: `ArchitectPromptPath(root, stage)` and `TicketPromptPath(root, type, stage)`
- Added stage constants: `StageSystem`, `StageKickoff`, `StageApprove`
- Changed `RenderTemplate` to accept `any` instead of `TicketVars` to support `ArchitectKickoffVars`

### Phase 4: Spawn Logic + API Handlers
- Orchestrator resolves agent config from ticket type instead of flat config
- Spawn functions use type-based prompt paths
- API handlers use `projectCfg.Architect.Agent/.Args` and type-based approve paths

### Phase 5: Install/Init
- New directory structure: `prompts/architect/` and `prompts/ticket/work/`
- Merged separate worktree prompt files into single templates using `{{if .IsWorktree}}` conditionals
- Updated config template to new nested schema

### Phase 6: Tests
- Updated all `store.Create` calls across 5 test files to pass 3rd ticketType argument
- Updated prompt path references from flat (`ticket-system.md`) to type-based (`ticket/work/SYSTEM.md`)
- Added new prompt path function tests

## Key Decisions
- **Empty Type defaults to "work"**: All code reading `ticket.Type` treats `""` as `DefaultTicketType` for backward compatibility with existing ticket JSON files
- **Breaking config change**: Projects must re-run `cortex init --force` to get the new config schema
- **Template `any` parameter**: Changed `RenderTemplate(content string, vars TicketVars)` to `RenderTemplate(content string, vars any)` to support both `TicketVars` and `ArchitectKickoffVars`
- **Unified worktree templates**: Instead of separate files for worktree/non-worktree, used Go template `{{if .IsWorktree}}` conditionals in single files

## Files Modified (25 total)
- `internal/project/config/config.go`, `config_test.go`
- `internal/ticket/ticket.go`, `store.go`, `store_test.go`
- `internal/types/response.go`, `convert.go`
- `internal/daemon/api/types.go`, `tickets.go`, `architect.go`, `sessions.go`, `integration_test.go`
- `internal/daemon/mcp/types.go`, `tools_architect.go`, `tools_ticket.go`, `tools_test.go`
- `internal/cli/sdk/client.go`
- `internal/prompt/prompt.go`, `prompt_test.go`, `template.go`
- `internal/core/spawn/orchestrate.go`, `spawn.go`, `spawn_test.go`
- `internal/install/install.go`, `prompts.go`

## Verification
- `go build ./...` — passes
- `go test ./...` — all tests pass
- `golangci-lint run` — 0 issues

## Follow-up Tasks
- Existing projects need `cortex init --force` to regenerate config and prompt files
- Consider adding migration logic or documentation for the config schema change
- Future ticket types (e.g., "investigation", "review") can now be added by defining them in the project config's `ticket` map and creating corresponding prompt directories