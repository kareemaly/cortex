---
id: 3b6f61f9-548d-48db-929e-2baf17c5e421
title: 'Debug: OpenCode architect spawn prints help and exits'
type: debug
tags:
    - opencode
    - debug
    - architect
created: 2026-02-12T11:31:29.585228Z
updated: 2026-02-12T11:43:08.120508Z
---
## Problem

When spawning an architect session on a project configured with `agent: opencode`, the launcher script runs `opencode` but it just prints the help/usage text and exits immediately instead of starting a session.

The project config is:
```yaml
name: radius
extend: ~/.cortex/defaults/opencode
ticket:
  work:
    agent: opencode
```

## Expected Behavior

OpenCode should launch in headless/run mode with the architect prompt, MCP tools, and model configured — similar to how `claude` agent sessions launch correctly.

## Investigation Areas

- Check how `buildOpenCodeCommand()` constructs the command for **architect** sessions (not just ticket sessions)
- The architect spawn path may not be routing through the opencode command builder at all
- Compare with how `buildClaudeCommand()` handles architect vs ticket sessions
- Check `internal/core/spawn/` for architect-specific launch logic
- Check if the opencode defaults `cortex.yaml` needs an `architect` section (currently only `ticket.work` may be defined)

## Acceptance Criteria

- OpenCode architect sessions launch correctly and enter an interactive session
- The architect prompt and MCP tools are properly injected