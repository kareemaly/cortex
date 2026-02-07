---
id: 18e289b7-a6a5-4d2b-8cfc-bb906c49faff
author: claude
type: done
created: 2026-01-29T12:40:38.283585Z
---
## Summary

Optimized embedded prompts for Claude Code agents and renamed the defaults folder from `basic` to `claude-code` to support future agent-specific configurations.

## Changes Made

### 1. Folder Rename (basic → claude-code)
- Renamed `setupBasicDefaults()` to `setupClaudeCodeDefaults()`
- Updated all path references from `~/.cortex/defaults/basic` to `~/.cortex/defaults/claude-code`
- Updated comments and config templates to reference new folder name

### 2. Architect SYSTEM.md Optimization
Reduced from 94 to 46 lines:
- Condensed role description to single sentence
- Shortened XML behavioral tags to essential guidance
- Replaced read/write tool categories with single inline tools list
- Added "State Transitions" section explaining automatic ticket moves
- Added "After Spawning" section with 4-step workflow clarification
- Removed "Writing Good Tickets" section (redundant)
- Kept minimal "Communication" section

### 3. Ticket SYSTEM.md Optimization
Reduced from 39 to 10 lines:
- Simplified to 5-tool list: `readTicket`, `addComment`, `addBlocker`, `requestReview`, `concludeSession`
- Reduced to 4-step workflow
- Removed detailed comments documentation (agent already knows)
- Removed context awareness section (duplicates Claude Code knowledge)

### 4. Ticket APPROVE.md Optimization
Reduced from 28 to 5 lines:
- Simplified to 3 steps: commit, push, conclude
- Removed worktree conditionals
- Removed verbose step-by-step instructions

## Files Modified

- `internal/install/install.go` - Folder rename + architect system prompt
- `internal/install/prompts.go` - Ticket system + approve prompts

## Rationale

The original prompts were written for a generic agent and included redundant information that Claude Code already knows. The optimized prompts are minimal, focusing only on Cortex-specific workflow while relying on Claude Code's built-in capabilities.

## Verification

- `make build` - Passed
- `make lint` - Passed (0 issues)
- `make test` - All tests pass

## Follow-up Notes

When `cortex init` is run, it will create `~/.cortex/defaults/claude-code/` instead of `basic/`. Existing installations with `basic/` will continue to work but won't receive the optimized prompts unless the user deletes the old folder and re-runs init with `--force`.