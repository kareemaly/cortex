---
id: 596ac796-8881-4acb-9fc6-03cc9ea4b60b
author: claude
type: ticket_done
created: 2026-01-26T17:53:53.627175Z
---
## Summary

Fixed the worktree approve prompt to render template variables before sending to the agent via tmux, and updated the prompt content to use local merge instead of push-and-PR workflow.

## Changes Made

### 1. internal/daemon/api/sessions.go
Added template rendering in the Approve handler. After loading the approve prompt file, the handler now builds a prompt.TicketVars struct (with project path, ticket ID/title/body, and optional worktree path/branch from the session) and calls prompt.RenderTemplate() to resolve Go template variables before sending the prompt to tmux. On render failure, logs a warning and falls through with unrendered content.

### 2. .cortex/prompts/approve-worktree.md
Replaced the prompt content: removed the push branch step, replaced with a local merge step using cd {{.ProjectPath}} && git merge {{.WorktreeBranch}}. Simplified from 4 steps to 3.

### 3. internal/install/prompts.go
Updated the DefaultApproveWorktreePrompt constant to match the new prompt content.

## Key Decisions
- Template rendering pattern mirrors existing pattern from internal/core/spawn/spawn.go for consistency
- On template render failure, logs warning but falls through with unrendered content
- Local merge over push matches the project workflow

## Files Modified
- internal/daemon/api/sessions.go (+21 lines)
- internal/install/prompts.go (+4/-8 lines)
- .cortex/prompts/approve-worktree.md (new file, 19 lines)

## Verification
- make build - compiles successfully
- make test - all tests pass
- make lint - 0 issues