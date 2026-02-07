---
id: 05c1eea1-a152-4b4a-ad3b-248b073b7557
title: 'Cleanup: Use Session.ID for Claude Session ID'
type: ""
created: 2026-01-23T07:29:06Z
updated: 2026-01-23T07:29:06Z
---
Remove `ClaudeSessionID` field and use `Session.ID` directly when spawning claude with `--session-id`.

## Rationale

We generate the session ID ourselves, so we can use the same ID for both our tracking and Claude's `--session-id` flag. No need for two separate IDs.

## Changes

- Remove `ClaudeSessionID` field from Session struct
- When spawning: `claude --session-id {session.ID} ...`
- When resuming: `claude --resume {session.ID} ...`

## Implementation

### Commits
- `d126c2f` refactor: remove ClaudeSessionID and use Session.ID directly
- `d8b0635` Merge branch 'ticket/2026-01-22-cleanup-session-id'

### Key Files Changed
- `internal/ticket/ticket.go` - Removed ClaudeSessionID from Session struct
- `internal/ticket/store.go` - Removed claudeSessionID parameter from SetSession
- `internal/core/spawn/command.go` - Added SessionID to ClaudeCommandParams for --session-id flag
- `internal/core/spawn/spawn.go` - Updated to pass session ID to BuildClaudeCommand, renamed ResumeRequest.ClaudeSessionID to SessionID
- `internal/core/spawn/state.go` - Removed ClaudeSessionID from StateInfo, updated CanResume logic
- `internal/daemon/api/types.go` - Removed ClaudeSessionID from SessionResponse
- `internal/daemon/mcp/types.go` - Removed ClaudeSessionID from SessionOutput
- `internal/cli/sdk/client.go` - Removed ClaudeSessionID from SDK SessionResponse

### Decisions
- Deleted obsolete test `TestHandleSpawnSession_StateOrphaned_ModeResumeNoSessionID` since every session now has an ID when created via SetSession, making the "no session ID" scenario impossible
- Used existing Session.ID directly for both --session-id (new spawns) and --resume (orphaned sessions)