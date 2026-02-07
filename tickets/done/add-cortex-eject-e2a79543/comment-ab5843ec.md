---
id: ab5843ec-9331-435f-b326-1838f30d1c2f
author: claude
type: review_requested
created: 2026-02-03T08:37:23.079833Z
action:
    type: git_diff
    args:
        commit: 17c4c9a
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/f67e420d-2e5f-47b0-84e7-5fcd55171ce4
---
## Summary

Added the `cortex eject` command that copies prompt files from the base configuration to the project's `.cortex/prompts/` directory for local customization.

## Changes

**New file: `cmd/cortex/commands/eject.go`**

The command:
1. Takes a prompt path as argument (e.g., `ticket/work/SYSTEM.md`)
2. Loads project config to get the resolved extend path
3. Validates the source file exists in the base prompts directory
4. Copies the file to the project's `.cortex/prompts/` directory
5. Supports `--force` flag to overwrite existing files

**Error handling:**
- Not in a cortex project: `not in a cortex project (no .cortex directory found)`
- No extend configured: `project has no 'extend' configured in .cortex/cortex.yaml`
- Source not found: `source prompt not found: {path}`
- Source is directory: `cannot eject directory: {path}`
- Destination exists: `destination already exists: {path}` (with hint to use --force)

**Success output:**
```
Ejected: ticket/work/SYSTEM.md
  From: ~/.cortex/defaults/claude-code/prompts/ticket/work/SYSTEM.md
  To:   .cortex/prompts/ticket/work/SYSTEM.md
```

## Verification

- `make build` - Passes
- `make lint` - Passes (fixed errcheck issues with proper Close() error handling)
- `cortex eject --help` - Shows correct usage and examples
- Tested error case when not in a project directory