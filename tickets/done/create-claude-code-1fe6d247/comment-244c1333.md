---
id: 244c1333-44e1-46db-9c63-b9ec92468cf1
author: claude
type: review_requested
created: 2026-02-03T08:37:34.510075Z
action:
    type: git_diff
    args:
        commit: e61e20b
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/c76ecbf8-3e87-4b9d-bceb-2059cb97c61f
---
Created `internal/install/defaults/claude-code/CONFIG_DOCS.md` (130 lines) with all sections from the plan:

1. **Project Config Schema** - YAML example and field descriptions for name, extend, architect, ticket, and git.worktrees
2. **Prompt Structure** - Table of all prompt files (SYSTEM, KICKOFF, APPROVE, REJECT for architect and ticket)
3. **Customizing Prompts** - Directory structure for overrides, resolution order (project → extend → defaults)
4. **Template Variables** - Tables for architect (ProjectName, TicketList) and ticket prompts (ProjectPath, TicketID, TicketTitle, TicketBody, IsWorktree, WorktreePath, WorktreeBranch)
5. **Common Customizations** - Examples for git worktrees, restricting permissions, adding test requirements, custom kickoff context
6. **Commands Reference** - Brief table for init, architect, kanban

Note: The plan mentioned `cortex eject` but that command doesn't exist yet. I documented the manual override mechanism (creating files in `.cortex/prompts/`) instead.