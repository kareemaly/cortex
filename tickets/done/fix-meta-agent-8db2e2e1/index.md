---
id: 8db2e2e1-a6b7-4abb-876a-e6d32de4739a
title: Fix meta agent companion pane to use cortex dashboard
type: chore
tags:
    - meta-agent
created: 2026-02-09T15:42:22.983539Z
updated: 2026-02-09T16:17:33.056093Z
---
The meta agent companion pane was set to `cortex projects` but should be `cortex dashboard`. Update the companion command in the meta spawn logic to use `cortex dashboard` instead.

Look in `internal/core/spawn/launcher.go` or `spawn.go` where the meta agent's companion pane command is configured.