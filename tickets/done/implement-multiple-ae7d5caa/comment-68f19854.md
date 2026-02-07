---
id: 68f19854-d282-4a7f-9d9f-98118b1d4c0c
author: claude
type: review_requested
created: 2026-02-05T07:01:29.131834Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Implemented multiple ticket types (debug, research, chore) with specialized workflows:

### New Ticket Types

| Type | Purpose | Key Behavior |
|------|---------|--------------|
| `debug` | Root cause analysis | Reproduce → Investigate → Document → Fix |
| `research` | Exploration | READ-ONLY mode, brainstorm, no file changes |
| `chore` | Maintenance | Minimal ceremony, direct execution |

### Changes Made

**1. Created 9 new prompt files:**
- `prompts/ticket/debug/{SYSTEM,KICKOFF,APPROVE}.md`
- `prompts/ticket/research/{SYSTEM,KICKOFF,APPROVE}.md`
- `prompts/ticket/chore/{SYSTEM,KICKOFF,APPROVE}.md`

**2. Updated `cortex.yaml`** with agent configurations for each type:
- debug: plan mode with addComment allowed
- research: plan mode (no skip-permissions for read-only)
- chore: dangerously-skip-permissions for quick execution

**3. Added type badge rendering to kanban:**
- `styles.go`: Added color styles (debug=red, research=blue, chore=gray)
- `column.go`: Prepends `[type]` badge to ticket titles for non-work types

**4. Added type validation in `tickets.go`:**
- Validates ticket type against project config on creation
- Returns clear error with valid types if invalid type specified

**5. Updated `embed_test.go`** to verify all 15 prompt files

**6. Updated `types.go`** MCP tool description with available types

### Verification

- `make build` ✓
- `make lint` ✓ (0 issues)
- `make test` ✓ (all tests pass)