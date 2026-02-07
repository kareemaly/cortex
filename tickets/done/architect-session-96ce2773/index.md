---
id: 96ce2773-4ab7-425a-b512-7c8425995435
title: Architect Session State
type: ""
created: 2026-01-22T13:19:48Z
updated: 2026-01-22T13:19:48Z
---
Create file-based state management for architect sessions, similar to ticket sessions.

## Context

This is a fresh project with no users. No backward compatibility needed. Breaking changes are fine. Do not accumulate tech debt.

## State File

Location: `.cortex/architect.json`

```json
{
  "id": "uuid",
  "tmux_session": "cortex-projectname",
  "tmux_window": "architect",
  "started_at": "2026-01-22T...",
  "ended_at": null
}
```

Note: `id` will be used as Claude's session ID for resume support (`claude --session-id`).

## Package Location

`internal/project/architect/` (or integrate into existing `internal/project/`)

## Functions Needed

- `Load(projectPath) → *ArchitectSession, error` - load state file, nil if not exists
- `Save(projectPath, session) → error` - write state file
- `Clear(projectPath) → error` - remove state file

## State Detection

Integrate with `internal/core/spawn/state.go` to detect architect session state:
- `Normal` - no state file or ended_at is set
- `Active` - state file exists, ended_at nil, tmux window exists
- `Orphaned` - state file exists, ended_at nil, tmux window gone
- `Ended` - state file exists, ended_at set

## Used By

- `internal/core/spawn/` - for architect spawning
- `internal/daemon/api/` - architect endpoints (future)
- `internal/daemon/mcp/` - architect tools (future)

## Verification

```bash
make lint
make test
make build
make test-integration
```

## Implementation

### Commits Pushed

- `67ec3e3` feat: add architect session state management with spawn state detection

### Key Files Changed

**Created:**
- `internal/project/architect/session.go` - Session struct with Load/Save/Clear functions
- `internal/project/architect/session_test.go` - 10 test cases for session management

**Modified:**
- `internal/core/spawn/state.go` - Added ArchitectStateInfo and DetectArchitectState
- `internal/core/spawn/spawn_test.go` - Added 7 architect state tests

### Important Decisions

- Used same state detection pattern as ticket sessions (Normal, Active, Orphaned, Ended)
- ArchitectStateInfo.CanResume() checks both Session != nil and Session.ID != "" (unlike ticket which uses ClaudeSessionID field)
- State file stored at `.cortex/architect.json` as specified
- Clear() is idempotent (no error if file doesn't exist)

### Scope Changes

None - implemented exactly as specified in the plan.