---
id: d4015bd8-5e03-4127-be0f-d3cdc5cbf26f
title: Add cortex config show CLI Command
type: work
created: 2026-02-02T11:35:57.316496Z
updated: 2026-02-02T11:45:32.71446Z
---
## Summary

Add a `cortex config show` CLI command that displays the fully resolved project configuration after all extends are merged. This helps users verify their config is correct and debug inheritance issues.

## Context

Users extending base configs (e.g., `extend: ~/.cortex/defaults/claude-code`) have no way to verify the final merged result. When ticket agents don't start with expected args (like plan mode), there's no visibility into whether the merge happened correctly.

## Requirements

### CLI Command
- `cortex config show` — displays resolved config for current project
- `cortex config show --path /some/project` — displays resolved config for specified project
- Output should be valid YAML that could be copy-pasted into a config file

### Output Format
```yaml
# Resolved config for: /path/to/project
# Extended from: ~/.cortex/defaults/claude-code

name: project-name
architect:
  agent: claude
  args:
    - "--allowedTools"
    - "mcp__cortex__listTickets,mcp__cortex__readTicket"
ticket:
  work:
    agent: claude
    args:
      - "--permission-mode"
      - "plan"
git:
  worktrees: false
```

### Implementation
- Use existing `projectconfig.Load()` which already handles extend resolution and merging
- Add comment header showing source path and extend chain
- Handle errors gracefully (missing config, invalid extend path, etc.)

## Acceptance Criteria
- [ ] `cortex config show` works in a project directory
- [ ] `cortex config show --path` works with explicit path
- [ ] Output shows fully merged config with all extends resolved
- [ ] Comment header shows extend chain for debugging
- [ ] Errors display helpful messages (no config found, extend path invalid, etc.)