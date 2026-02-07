---
id: 03d2f382-af2a-4e44-9cf8-1fe80f2f5f8c
author: claude
type: ticket_done
created: 2026-01-28T07:44:28.469638Z
---
## Summary

Updated three ticket prompt files to improve the agent workflow with explicit push steps, a new comment type, and context awareness guidance.

## Changes Made

### 1. `.cortex/prompts/approve-worktree.md`
- Added step 3 "Push to origin" (`cd {{.ProjectPath}} && git push`) between the merge and concludeSession steps
- Renumbered concludeSession from step 3 to step 4

### 2. `.cortex/prompts/approve.md`
- Changed step 2 heading from "Push your branch" to "Push to origin"
- Changed command from `git push` to `git push origin HEAD` for explicit remote and ref targeting

### 3. `.cortex/prompts/ticket-system.md`
- Added `scope_change` to the comment types in the tool description line
- Added `scope_change` entry in the Comments section list: "Changes to the ticket scope or requirements"
- Added new "Context Awareness" section after "Important" with three bullet points:
  - Context window compaction warning
  - Frequent commit guidance
  - Using `addTicketComment` with `progress` type for milestone logging and context recovery

## Files Modified
- `.cortex/prompts/approve-worktree.md`
- `.cortex/prompts/approve.md`
- `.cortex/prompts/ticket-system.md`

## Decisions
- All changes were prompt-only, no build or test verification needed
- Followed the exact plan specifications without deviation

## Follow-up
- The `scope_change` comment type must also be supported in the backend (`addTicketComment` tool handler) — verify it's already accepted or add it if not