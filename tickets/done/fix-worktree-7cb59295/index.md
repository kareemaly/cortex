---
id: 7cb59295-9601-4a96-8e50-4c3339570a8a
title: Fix Worktree Approve Prompt and Template Rendering
type: ""
created: 2026-01-26T17:42:37.715526Z
updated: 2026-01-26T17:53:53.628367Z
---
## Problem

Two issues with the worktree approve flow:

1. **Template variables not rendered**: The `Approve` handler in `internal/daemon/api/sessions.go` loads the prompt file but never calls `prompt.RenderTemplate()`, so `{{.WorktreeBranch}}` is sent as literal text to the agent.

2. **Wrong instructions**: The approve-worktree prompt tells the agent to push the branch remotely. It should merge the worktree branch into main locally instead.

## Fixes Required

### 1. Fix Template Rendering in Approve Handler

In `internal/daemon/api/sessions.go`, after loading the approve prompt content, call `prompt.RenderTemplate(approveContent, vars)` before sending to tmux. The ticket and session data needed for `TicketVars` is already available in the handler.

### 2. Update `.cortex/prompts/approve-worktree.md`

Replace with:

```markdown
## Review Approved

Your changes have been reviewed and approved. Complete the following steps:

1. **Commit all changes**
   - Run `git status` to check for uncommitted changes
   - Commit any remaining changes

2. **Merge to main**
   - Run `cd {{.ProjectPath}} && git merge {{.WorktreeBranch}}`

3. **Call concludeSession**
   - Call `mcp__cortex__concludeSession` with a complete report including:
     - Summary of all changes made
     - Key decisions and their rationale
     - List of files modified
     - Any follow-up tasks or notes

This will mark the ticket as done and end your session.
```

### 3. Update Default Prompt in Install Command

The install command embeds default prompts. Update the embedded approve-worktree template to match the new content above. Search for the default approve-worktree content in the install/setup code.

## Key Files

| File | Change |
|------|--------|
| `internal/daemon/api/sessions.go` | Call `prompt.RenderTemplate()` before sending approve content |
| `.cortex/prompts/approve-worktree.md` | Update prompt text (merge locally, not push) |
| Install command (embedded defaults) | Update default approve-worktree template to match |

## Acceptance Criteria

- [ ] Approve handler renders template variables before sending to agent
- [ ] `{{.WorktreeBranch}}` and `{{.ProjectPath}}` are substituted correctly
- [ ] Worktree approve prompt instructs agent to merge locally, not push
- [ ] Default embedded prompt in install command matches updated template