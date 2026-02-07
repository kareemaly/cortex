---
id: 469749fb-17cb-4a9e-ba4b-c886b89a22dc
author: claude
type: comment
created: 2026-02-07T10:29:31.999653Z
---
## Audit Finding: CLEAN — Spawn Orchestration

**Files:** orchestrate.go, state.go, spawn.go, launcher.go, spawn_test.go

### State Machine:
- **3 states**: StateNormal (no session), StateActive (session + tmux window), StateOrphaned (session but no window)
- **3 modes**: normal, resume, fresh
- **State x Mode matrix**: All 9 combinations handled correctly:
  - Normal+normal → Spawn new
  - Normal+resume/fresh → Error (nothing to resume/replace)
  - Active+normal → AlreadyActive error
  - Active+resume/fresh → Error (still active)
  - Orphaned+normal → Error (must choose resume or fresh)
  - Orphaned+resume → Resume existing
  - Orphaned+fresh → End old session, spawn fresh

### Session Store Integration:
- Spawn(): Creates session via SessionStore.Create() with ticketID, agent, windowName, worktreePath, featureBranch
- Resume(): Retrieves session via GetByTicketID, passes existing tmux window name
- Fresh(): Calls SessionStore.End() to cleanup, then spawns new
- cleanupOnFailure(): Calls SessionStore.End() if ticket agent spawn fails
- DetectTicketState(): Uses GetByTicketID to check for existing session

### Resume Flag Generation:
- Verified correct at launcher.go:131-135
- `ResumeID != ""` → `--resume <id>`
- `Resume == true` && `ResumeID == ""` → bare `--resume`
- spawn.go:368-369 correctly sets Resume=true when SessionID empty, ResumeID=SessionID string

### Test Coverage:
- spawn_test.go: 672 lines, comprehensive coverage of all state/mode combinations
- StateEnded correctly removed (comment at tools_test.go:872 explains why)

**Severity: CLEAN**