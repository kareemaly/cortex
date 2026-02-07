---
id: 3883fa7b-615e-4c19-8901-3a282b9f31b0
title: 'Update Ticket Prompts: Add Push to Origin, Fix Comment Types, Add Context Awareness'
type: ""
created: 2026-01-28T07:41:23.158993Z
updated: 2026-01-28T07:44:28.470948Z
---
## Summary

Update the ticket agent prompts in `.cortex/prompts/` for the cortex1 project. These are prompt files only — no Go code changes.

## Changes

### 1. `approve-worktree.md` — Add push to origin after merge

Add a step to push to origin HEAD after merging the worktree branch to main. Current flow stops at merge. Updated flow:

1. Commit all changes
2. Merge to main (`cd {{.ProjectPath}} && git merge {{.WorktreeBranch}}`)
3. **Push to origin** (`cd {{.ProjectPath}} && git push`)
4. Call concludeSession

### 2. `approve.md` — Make push target explicit

Currently says "push your branch" with `git push`. Make it explicit that the agent should push to origin HEAD after committing.

### 3. `ticket-system.md` — Add missing comment type and context awareness

- Add `scope_change` to the list of comment types (currently missing, but supported by the system)
- Add a context awareness note so ticket agents know their context window may be compacted and they should commit work incrementally

## Scope

These changes only affect `.cortex/prompts/` files in the current cortex1 project. No changes to Go code, init templates, or other projects.

## Acceptance Criteria

- `approve-worktree.md` includes push to origin step after merge and before concludeSession
- `approve.md` explicitly pushes to origin
- `ticket-system.md` lists `scope_change` as a comment type
- `ticket-system.md` includes context awareness guidance