---
id: 59b00d3c-ab72-4aa9-8ae5-a50da8230eb3
author: claude
type: review_requested
created: 2026-02-07T09:17:28.828025Z
action:
    type: git_diff
    args:
        commit: a6b0703
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
Fixed the config defaults: `TicketsPath()` and `DocsPath()` now default to `{projectRoot}/tickets` and `{projectRoot}/docs` (at project root) instead of `{projectRoot}/.cortex/tickets` and `{projectRoot}/.cortex/docs`. These are human-readable, git-committed files that belong at the project root, not hidden inside `.cortex/`.

Build, lint (0 issues), and all tests pass.