---
id: f8ebd9a6-47b9-4b02-ab3c-a065adbdf585
title: Fix Diff Action to CD into Repo Path Before Running
type: work
created: 2026-02-02T16:25:12.649919Z
updated: 2026-02-02T16:40:26.945342Z
---
## Summary

The git_diff action opens lazygit/git diff but runs from project root instead of the specified repo path. Projects with multiple repos need the command to run from the correct repo directory.

## Problem

Current behavior:
- Action has `repo_path: /full/path/to/repo`
- Tmux popup opens with `-d "<repo_path>"` flag
- But lazygit/git may not respect working directory correctly

## Fix

Change the tmux popup command to explicitly cd first:

```bash
cd /full/path/to/repo && lazygit
# or
cd /full/path/to/repo && git diff <commit>
```

## Location

`internal/daemon/api/tickets.go` — `ExecuteAction` handler

## Acceptance Criteria
- [ ] Diff action runs `cd <repo_path> && <command>`
- [ ] Works correctly for multi-repo projects