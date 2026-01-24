# Simplify Architect Session

## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Problem

Architect session tracking with architect.json is overcomplicated. We don't need --session-id, --fresh, or --resume for architect.

## Requirements

- Remove architect.json session tracking
- Never pass --session-id to architect claude command
- Simple logic: if tmux window exists → focus it, otherwise → spawn fresh
- Remove --fresh and --resume options from `cortex architect` command

## Implementation

### Commits Pushed
- `7d60756` feat: simplify architect session to window-based detection

### Key Files Changed
- `cmd/cortex/commands/architect.go` - Removed `--fresh` and `--resume` flags, simplified to always pass empty mode
- `internal/daemon/api/architect.go` - Rewrote spawn logic: check window exists → focus or spawn fresh
- `internal/core/spawn/state.go` - Removed `ArchitectStateInfo`, `DetectArchitectState()` and related methods
- `internal/core/spawn/spawn.go` - Updated `Resume()` to only support ticket agents (added TicketID validation)
- `internal/core/spawn/spawn_test.go` - Removed all architect state tests, added `TestResume_NoTicketID`

### Files Deleted
- `internal/project/architect/session.go` - Session tracking no longer needed
- `internal/project/architect/session_test.go` - Associated tests

### Important Decisions
- Kept ticket session state detection intact (tickets still need orphan detection and resume)
- Resume functionality now explicitly requires TicketID (only ticket agents can resume)
- Architect spawn response still includes session info for API compatibility, but ID is always empty

### Scope Changes
None - implemented as specified
