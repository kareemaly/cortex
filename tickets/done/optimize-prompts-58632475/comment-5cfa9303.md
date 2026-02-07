---
id: 5cfa9303-1adf-45d0-9dbf-2e12ac124306
author: claude
type: review_requested
created: 2026-01-29T11:40:23.071742Z
action:
    type: git_diff
    args:
        commit: "1281434"
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Optimized embedded prompts for Claude Code agents and renamed the defaults folder from `basic` to `claude-code`.

### Changes Made

**1. Folder rename (basic → claude-code)**
- `setupBasicDefaults()` → `setupClaudeCodeDefaults()` 
- Updated all references in comments and config values
- `extend: ~/.cortex/defaults/basic` → `extend: ~/.cortex/defaults/claude-code`

**2. Architect SYSTEM.md (94 → 46 lines)**
- Condensed role description to single sentence
- Shortened XML behavioral tags
- Simplified to single tools list instead of read/write categories
- Added "State Transitions" section explaining auto-moves
- Added "After Spawning" section with 4-step workflow
- Removed "Writing Good Tickets" section
- Removed "Communication Style" (kept minimal version)

**3. Ticket SYSTEM.md (39 → 10 lines)**
- Reduced to 4-tool list: `readTicket`, `addComment`, `addBlocker`, `requestReview`, `concludeSession`
- Simplified to 4-step workflow
- Removed detailed comments documentation
- Removed context awareness section (duplicates what Claude Code already knows)

**4. Ticket APPROVE.md (28 → 5 lines)**
- Reduced to 3 simple steps: commit, push, conclude
- Removed worktree conditionals
- Removed verbose instructions

### Files Modified

- `internal/install/install.go` - Folder rename + architect system prompt
- `internal/install/prompts.go` - Ticket system prompt + approve prompt

### Verification

- `make build` - Passed
- `make lint` - Passed (0 issues)
- `make test` - All tests pass