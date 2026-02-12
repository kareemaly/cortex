---
id: dc865a55-be38-4acd-9825-8c218c30cc25
author: claude
type: review_requested
created: 2026-02-09T15:43:07.346033Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/a00e6bce-5ebc-4410-8c06-22af9f1d6f11
        commit: ec825de
---
Changed meta agent companion pane command from `cortex projects` to `cortex dashboard` in both spawn code paths in `internal/core/spawn/spawn.go` (lines 484 and 911). Build verified.