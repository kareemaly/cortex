---
id: 0e866876-851e-4fdf-832b-5862f346c875
author: claude
type: done
created: 2026-02-03T08:40:44.308953Z
---
## Summary

Created `internal/install/defaults/claude-code/CONFIG_DOCS.md` (130 lines) - configuration documentation for the claude-code agent type.

### Document Sections:
1. **Project Config Schema** - YAML example and field descriptions for name, extend, architect, ticket, and git.worktrees
2. **Prompt Structure** - Table of all prompt files (SYSTEM, KICKOFF, APPROVE, REJECT)
3. **Customizing Prompts** - Directory structure for overrides, resolution order (project → extend → defaults)
4. **Template Variables** - Tables for architect prompts (ProjectName, TicketList) and ticket prompts (ProjectPath, TicketID, TicketTitle, TicketBody, IsWorktree, WorktreePath, WorktreeBranch)
5. **Common Customizations** - Examples for git worktrees, restricting permissions, test requirements, custom kickoff
6. **Commands Reference** - init, architect, kanban

### Additional Change:
Updated `internal/install/defaults/claude-code/prompts/ticket/work/APPROVE.md` to include conditional worktree merge instructions using template variables.

### Commits:
- `e61e20b` - docs: add claude-code CONFIG_DOCS.md
- `8733936` - feat: add worktree merge instructions to APPROVE.md

### Merged and pushed to origin/main.