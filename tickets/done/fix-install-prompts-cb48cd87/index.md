---
id: cb48cd87-fd4c-419b-a8e5-14d960da9868
title: Fix Install Prompts
type: ""
created: 2026-01-24T15:41:03Z
updated: 2026-01-24T15:41:03Z
---
## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

`cortex install` creates legacy prompt files instead of the new v2 prompt structure.

**Current (wrong):**
- `architect.md`
- `ticket-agent.md`

**Should be:**
- `architect.md`
- `ticket-system.md` - MCP tool instructions (appended to system prompt)
- `ticket.md` - Non-worktree ticket prompt
- `ticket-worktree.md` - Worktree ticket prompt
- `approve.md` - Non-worktree approve prompt
- `approve-worktree.md` - Worktree approve prompt

Also missing `tickets/review` folder.

## Requirements

- Update `internal/install/` to create the 5 new prompt files
- Remove legacy `ticket-agent.md` creation
- Add `tickets/review` folder creation

Another ticket please also work on

# Remove Install Project Parameter

## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

`cortex install` has a `--project` parameter but it should always run from inside a project folder. The parameter is unnecessary.

Also there may be test failures or error messages that reference `cortex install --project ...` which need updating.

## Requirements

- Remove `--project` flag from `cortex install` command
- Install always uses current working directory as project path
- Update any error messages, tests, or docs that reference `--project`

## Implementation

### Commits
- `b65f5a9` fix: remove legacy ticket-agent.md support and --project flag references

### Key Files Changed
- `internal/install/install.go` - Removed `defaultTicketAgentPrompt` constant and legacy file creation
- `internal/prompt/prompt.go` - Deleted deprecated `TicketAgentPath` function
- `internal/prompt/errors.go` - Updated error message from `cortex install --project` to `cortex install`
- `internal/core/spawn/spawn.go` - Removed fallback logic to legacy `ticket-agent.md`
- `internal/prompt/prompt_test.go` - Removed `TestTicketAgentPath` test, updated error message assertion
- `internal/core/spawn/spawn_test.go` - Updated test setup to use `ticket-system.md`
- `internal/daemon/mcp/tools_test.go` - Updated test setup to use `ticket-system.md`

### Decisions
- The install already creates all 6 v2 prompt files correctly; only needed to remove the legacy `ticket-agent.md` creation
- The `--project` flag doesn't exist in the current CLI implementation; only needed to update error messages and tests that still referenced it
- The `tickets/review/` folder was already being created in `setupProject`

### Scope
- Narrower than original ticket: Install was already creating v2 prompts, just had legacy code that also created `ticket-agent.md`
- The `--project` flag removal was already done; this ticket cleaned up remaining references