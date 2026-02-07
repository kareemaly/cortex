---
id: a78a2e07-7061-48fc-8dcf-4579af8224aa
title: Add Agent CLI Args Configuration to cortex.yaml
type: ""
created: 2026-01-26T18:02:37.873668Z
updated: 2026-01-26T18:09:28.915557Z
---
## Problem

There's no way to configure agent CLI flags per-project. For example, passing `--model` to select a specific model or `--dangerously-skip-permissions` to skip permission prompts for trusted projects.

## Solution

Add an `agent` config section in `cortex.yaml` with a list of CLI args passed to the agent command when spawning ticket sessions.

### Config Format

```yaml
name: cortex
agent: claude
agent_args:
  - "--model"
  - "sonnet"
  - "--dangerously-skip-permissions"
git:
  worktrees: false
```

### Implementation

1. **Add config field** in `internal/project/config/config.go`:
   - Add `AgentArgs []string` field to `Config` struct

2. **Pass args to spawn**:
   - Thread `AgentArgs` through `Orchestrate()` → `SpawnRequest` → launcher script
   - Append args to the claude CLI command in the launcher script

3. **Update launcher script generation** in `internal/core/spawn/spawn.go`:
   - When building the claude command, append the configured args

## Key Files

| File | Change |
|------|--------|
| `internal/project/config/config.go` | Add `AgentArgs` field to Config |
| `internal/core/spawn/orchestrate.go` | Read and pass agent args |
| `internal/core/spawn/spawn.go` | Append args to claude CLI command in launcher |

## Acceptance Criteria

- [ ] `agent_args` list in cortex.yaml is parsed correctly
- [ ] Args are appended to the agent CLI command on spawn
- [ ] Works with `--model`, `--dangerously-skip-permissions`, and any other CLI flags
- [ ] Empty or missing `agent_args` has no effect (backwards compatible)