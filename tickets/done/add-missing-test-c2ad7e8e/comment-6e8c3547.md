---
id: 6e8c3547-ca55-45a5-8368-3d3cf265f7c1
author: claude
type: review_requested
created: 2026-02-08T13:28:13.205515Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/083b081b-7b7b-4e38-b9da-cb1abd878457
        commit: de159b7
---
## Summary

Added comprehensive unit test coverage to three previously untested high-risk packages:

### 1. `internal/cli/sdk/client_test.go` — 52 tests
- **Helpers**: `routedServer` with method+path routing for multi-step SDK methods (e.g., UpdateTicket calls GetTicketByID then PUT)
- **Constructor/Config**: NewClient, DefaultClient, WithProject (sharing HTTP client, empty path returns same client)
- **Project header**: doRequest injects X-Cortex-Project header, empty path omits it
- **Global endpoints**: Health, HealthWithVersion (success, server error, connection error, bad JSON), ListProjects, UnlinkProject, FocusDaemonDashboard
- **Ticket CRUD**: ListAllTickets (no filters, with query/due_before/tag filters), ListTicketsByStatus, GetTicket (success, not found), GetTicketByID, CreateTicket (basic, all fields, error), UpdateTicket, DeleteTicket, MoveTicket
- **Due dates**: SetDueDate, ClearDueDate
- **Sessions**: SpawnSession, KillSession, ApproveSession, ListSessions
- **Architect**: GetArchitect, SpawnArchitect, ConcludeArchitectSession
- **Comments/Reviews**: AddComment, RequestReview (verifies request body), ConcludeSession, ExecuteCommentAction
- **Docs**: CreateDoc, GetDoc, UpdateDoc, DeleteDoc, MoveDoc, ListDocs (query params), AddDocComment
- **Error parsing**: WithDetails (Details takes precedence), MalformedJSON (raw body in message), APIError.IsOrphanedSession
- **Focus**: FocusArchitect, FocusTicket, ResolvePrompt (query params)
- **hasPrefix**: table-driven tests for short ID matching

### 2. `internal/upgrade/upgrade_test.go` — 22 tests
- **Version comparison** (table-driven): equal, a<b, a>b, minor/patch diffs, v-prefix handling, prerelease stripped, partial versions
- **shouldUpgrade** (table-driven): dev always upgrades, same version no upgrade, newer/older/major/patch
- **ParseChecksums** (table-driven): two-space format, single-space fallback, empty lines, empty content, malformed lines
- **VerifyChecksum**: success (compute SHA256), mismatch, file not found
- **Download**: success (httptest.Server serves file), server error (500)
- **DownloadString**: success, server error (404)
- **GetAssetURL**: found, not found
- **GetDownloadBaseURL**: verify URL format
- **GetBinaryName**: verify `{name}-{GOOS}-{GOARCH}` format
- **BackupBinary**: success (verify content and naming), source not found
- **ReplaceBinary**: no-sudo path (verify file contents replaced)
- **CleanupBackups**: keeps recent (5 backups, keep 3, verify oldest removed), nothing to clean (<keepCount), nonexistent dir (returns nil)
- **GetBackupDir**: verify path ends with `.cortex/backups`

### 3. `internal/daemon/api/tickets_test.go` — 33 tests
- **Setup**: `unitServer` with own helpers (independent of build-tagged integration_test.go), creates `.cortex/cortex.yaml` for ProjectRequired middleware
- **SetDueDate**: success, invalid JSON, empty due_date, invalid format, not found
- **ClearDueDate**: success, not found
- **GetByID**: success (verifies status), not found
- **AddComment**: success (with explicit author), invalid JSON, invalid type, ticket not found, all valid types (table-driven: comment, review_requested, done, blocker)
- **RequestReview**: success, invalid JSON, empty repo_path, empty content, moves ticket to review status
- **Conclude**: success (verifies ticket moves to done), invalid JSON, empty content, not found
- **Focus**: no session manager, no active session
- **List filters**: due_before invalid format, query filter, tag filter
- **Create edge cases**: with due_date, invalid due_date format
- **ListByStatus**: due_before invalid format
- **validStatus**: table-driven for all valid/invalid statuses

All 107 tests pass. `make test` passes clean.