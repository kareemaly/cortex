---
id: 214fb227-d0b1-47f6-87ad-62fe8a7cadeb
title: Ticket Workflow Refactor
type: ""
created: 2026-01-22T10:07:19Z
updated: 2026-01-22T10:07:19Z
---
Refactor ticket schema and MCP tools to support review status, incremental comments, and flexible lifecycle hooks.

## Schema Changes

### Statuses
Add `review` status: `backlog → progress → review → done`

### Dates
```go
type Dates struct {
    Created  time.Time
    Updated  time.Time
    Progress *time.Time  // when first moved to progress
    Reviewed *time.Time  // when moved to review
    Done     *time.Time  // when moved to done
}
```

### Comments (new)
```go
type CommentType string
const (
    CommentScopeChange CommentType = "scope_change"
    CommentDecision    CommentType = "decision"
    CommentBlocker     CommentType = "blocker"
    CommentProgress    CommentType = "progress"
    CommentQuestion    CommentType = "question"
    CommentRejection   CommentType = "rejection"  // user-added when respawning
    CommentGeneral     CommentType = "general"
)

type Comment struct {
    ID        string
    SessionID string      // empty if user-added
    Type      CommentType
    Content   string
    CreatedAt time.Time
}
```

Comments live on Ticket (not Session).

### Session
Remove `Report` and `GitBase`. Keep `StatusEntry` fields as placeholder.

```go
type Session struct {
    ID            string
    StartedAt     time.Time
    EndedAt       *time.Time
    Agent         string
    TmuxWindow    string
    CurrentStatus *StatusEntry
    StatusHistory []StatusEntry
}
```

## MCP Ticket Session Tools

Replace current tools (`pickupTicket`, `submitReport`, `approve`) with:

| Tool | Input | Description |
|------|-------|-------------|
| `readTicket` | *(none)* | Read assigned ticket |
| `moveTicketToProgress` | *(none)* | Move to progress, run hook |
| `moveTicketToReview` | *(none)* | Move to review, run hook |
| `moveTicketToDone` | *(none)* | Move to done, run hook |
| `addTicketComment` | `type`, `content` | Add comment, run hook |
| `concludeSession` | *(none)* | End session, run hook |

All tools return `{success, hooks_output}` where `hooks_output` is raw stdout from the hook command.

## Lifecycle Hooks

```yaml
lifecycle:
  moved_to_progress: <command>
  moved_to_review: <command>
  moved_to_done: <command>
  comment_added: <command>
  session_ended: <command>
```

Template variables available:
- `{{.TicketID}}`, `{{.Slug}}`, `{{.Title}}`, `{{.Body}}`
- `{{.CommentType}}`, `{{.Comment}}` (for comment_added)
- `{{.SessionID}}`, `{{.Agent}}`

Hooks are informational - raw output returned to agent who reads and reacts.

## Store Changes

- Create `review/` directory alongside backlog/progress/done
- Add `AddComment(ticketID, sessionID, commentType, content)` method
- Update `Move()` to set appropriate date field
- Remove `UpdateSessionReport()`

## Kanban TUI

Add 4th column for Review status.

## Files Affected

- `~/projects/cortex1/internal/ticket/ticket.go` - schema
- `~/projects/cortex1/internal/ticket/store.go` - new methods
- `~/projects/cortex1/internal/daemon/mcp/tools_ticket.go` - new tools
- `~/projects/cortex1/internal/daemon/mcp/types.go` - new I/O types
- `~/projects/cortex1/internal/project/config/config.go` - new hook types
- `~/projects/cortex1/internal/lifecycle/` - new hook execution
- `~/projects/cortex1/internal/cli/tui/kanban/` - 4th column
- `~/projects/cortex1/internal/daemon/api/` - update handlers for new status

## Implementation

### Commits Pushed
- `7391797` feat: add review status and comment-based ticket workflow

### Key Files Changed
- `internal/ticket/ticket.go` - Added StatusReview, CommentType, Comment struct, updated Dates with Progress/Reviewed/Done fields, simplified Session by removing GitBase and Report
- `internal/ticket/store.go` - Added review status support, AddComment method, updated Move to set date fields, changed AddSession signature
- `internal/daemon/mcp/tools_ticket.go` - Complete rewrite with new tools (moveTicketToProgress, moveTicketToReview, moveTicketToDone, addTicketComment, concludeSession) plus deprecated backward-compatible wrappers
- `internal/daemon/mcp/types.go` - Added AddCommentInput/Output, ConcludeSessionInput/Output, CommentOutput, updated DatesOutput and SessionOutput
- `internal/project/config/config.go` - Added new lifecycle hook configurations
- `internal/lifecycle/hooks.go` - Added new hook type constants
- `internal/lifecycle/template.go` - Extended TemplateVars with TicketBody, SessionID, Agent, CommentType, Comment; added WithSession, WithComment methods
- `internal/cli/tui/kanban/model.go` - Changed from 3 to 4 columns, added review column handling
- `internal/cli/tui/kanban/styles.go` - Added review column styling (blue color)
- `internal/daemon/api/types.go` - Added Review to ListAllTicketsResponse, updated date/comment/session structures
- `internal/cli/sdk/client.go` - Added Review field, updated FindTicketByID to search review status

### Important Decisions
- Deprecated old tools (pickupTicket, submitReport, approve) but kept them for backward compatibility - they delegate to new implementations
- Comments are stored at ticket level, not session level, allowing persistence across sessions
- All MCP tools now return hooks_output field with raw stdout from hook execution
- Session no longer tracks GitBase or Report - simplified to just status tracking

### Scope Changes
- None - implementation followed the original ticket spec exactly