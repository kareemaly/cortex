---
id: fb56ebc2-8301-4c93-8a86-5b7ecd221c9b
title: Rewrite spawn orchestration for session store
type: work
created: 2026-02-07T09:19:07.430296Z
updated: 2026-02-07T09:32:02.066119Z
---
## Overview

Ticket 2a made the daemon compile with the new storage layer, but the spawn orchestration (`internal/core/spawn/`) was only patched to compile — not properly rewritten. This ticket rewrites spawn to fully leverage the independent session store.

## Context

Sessions are now ephemeral and independent from tickets:
- Stored in `.cortex/sessions.json` (per project)
- Keyed by session ID
- `session.Store` provides: `Create`, `Get`, `GetByTicketID`, `UpdateStatus`, `End`, `List`
- No session data on tickets — `ticket.Session` is gone
- When a session ends, it's deleted (not marked as ended)

The spawn package already has `SessionStoreInterface` from ticket 2a:
```go
type SessionStoreInterface interface {
    Create(ticketID, agent, tmuxWindow string, worktreePath, featureBranch *string) (string, *session.Session, error)
    End(ticketShortID string) error
    GetByTicketID(ticketID string) (*session.Session, error)
}
```

## What Needs Rewriting

### State Detection (`state.go`)

The old model had 4 states: `normal`, `active`, `orphaned`, `ended`. With ephemeral sessions, `ended` is gone — if there's no session, it's `normal`.

New state model:
- **normal**: No active session exists → can spawn fresh
- **active**: Session exists AND tmux window alive → session is running
- **orphaned**: Session exists BUT tmux window gone → stale session entry

`DetectTicketState` should:
1. Look up session via `GetByTicketID(ticketID)`
2. If no session → `normal`
3. If session exists, check tmux window → `active` or `orphaned`

### Spawn Flow (`spawn.go` / `orchestrate.go`)

Verify and clean up the spawn flow:
1. **Normal spawn**: Create session in session store, spawn tmux window
2. **Resume**: Reattach to existing tmux window (session already exists)
3. **Fresh**: End existing session, create new one, spawn new tmux window

Ensure:
- `SessionStore.Create()` is called at the right point (after tmux window creation succeeds)
- `SessionStore.End()` is called on cleanup/fresh spawn
- Error paths clean up session entries
- The orchestrator passes session store correctly through all layers

### Tests (`spawn_test.go`)

Ticket 2a already updated tests to use split mock stores. Verify tests are comprehensive:
- All 3 state detections (normal, active, orphaned)
- All 3 spawn modes (normal, resume, fresh)
- Error paths and cleanup
- Session store interactions verified via mock

## Goals

- Spawn orchestration fully uses session store (no hacks or stubs)
- State detection is clean with 3 states (no `ended`)
- All spawn tests pass
- `make build && make lint && make test` pass

## Branch

Working on `feat/frontmatter-storage` branch.