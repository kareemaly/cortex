# Ticket Workflow V2

## Context

Early development, no users. Breaking changes are fine. Do not accumulate tech debt.

## Overview

Complete overhaul of ticket agent workflow. Replace lifecycle hooks and move tools with simpler requestReview/concludeSession flow.

---

## Part 1: Cleanup

**Remove lifecycle hooks entirely:**
- Delete `internal/lifecycle/` package
- Remove lifecycle config from `.cortex/cortex.yaml` schema
- Remove hook execution from all MCP tools

**Remove ticket move tools:**
- Delete `moveTicketToProgress`, `moveTicketToReview`, `moveTicketToDone` from `tools_ticket.go`
- Keep `readTicket` and `addTicketComment` (remove hook execution from addTicketComment)

---

## Part 2: New MCP Tools + Schema

**Session schema update:**
```go
type Session struct {
    // ... existing
    RequestedReviews []ReviewRequest `json:"requested_reviews"`
}

type ReviewRequest struct {
    RepoPath    string    `json:"repo_path"`
    Summary     string    `json:"summary"`
    RequestedAt time.Time `json:"requested_at"`
}
```

**New ticket agent tools:**

`requestReview` - Agent calls when work is ready for review
- Input: `repo_path` (string), `summary` (string)
- Appends to `Session.RequestedReviews`
- Can be called multiple times (multi-repo)

`concludeSession` - Agent calls to end session (from approve prompt)
- Input: `full_report` (string - commits, pushed branches, key decisions, etc.)
- Stores report as `ticket_done` comment on the ticket
- Moves ticket to done
- Ends session (sets `ended_at`)

---

## Part 3: Prompt Templates

Create 5 files in `.cortex/prompts/`:

**ticket-system.md** (appended via `--append-system-prompt`)
- Instructions to call `mcp__cortex__requestReview` after finalizing work
- Can call multiple times for multi-repo
- Use `mcp__cortex__addTicketComment` for decisions/blockers

**ticket.md** (prompt for non-worktree spawn)
- Template vars: `{{.ProjectPath}}`, `{{.TicketID}}`, `{{.TicketTitle}}`, `{{.TicketBody}}`
- User-customizable (git checkout instructions, etc.)

**ticket-worktree.md** (prompt for worktree spawn)
- Same as ticket.md + `{{.WorktreePath}}`, `{{.WorktreeBranch}}`
- Says "you are in a worktree"

**approve.md** (prompt for non-worktree approve)
- Instructions to commit, push
- Call `mcp__cortex__concludeSession` with full report

**approve-worktree.md** (prompt for worktree approve)
- Merge worktree branch to main
- Push
- Call `mcp__cortex__concludeSession`

---

## Part 4: Spawn Flow Update

Update spawn to use new prompts:
- Prompt: render `ticket.md` or `ticket-worktree.md`
- Add `--append-system-prompt` with `ticket-system.md` content
- Install prompt files via `cortex init` (error if missing)

---

## Part 5: Approve Flow

When user triggers approve (TUI or CLI):
1. Render `approve.md` or `approve-worktree.md`
2. Send content to agent's tmux pane via keystroke
3. Agent commits/pushes/merges, calls `concludeSession`
4. `concludeSession` moves ticket to done, ends session

---

## Implementation

### Commits Pushed
- `5e2a240` feat: implement ticket workflow v2 with requestReview/concludeSession flow

### Key Files Changed

**Deleted:**
- `internal/lifecycle/` (entire package - hooks.go, template.go, errors.go, hooks_test.go)
- `internal/daemon/mcp/helpers.go`

**New files:**
- `internal/install/prompts.go` - Default prompt templates
- `internal/prompt/template.go` - Template rendering with TicketVars

**Modified:**
- `internal/project/config/config.go` - Removed LifecycleConfig
- `internal/ticket/ticket.go` - Added ReviewRequest and RequestedReviews to Session
- `internal/ticket/store.go` - Added AddReviewRequest method
- `internal/daemon/mcp/types.go` - Added RequestReviewInput/Output, ConcludeSessionInput/Output
- `internal/daemon/mcp/tools_ticket.go` - Replaced move tools with requestReview/concludeSession
- `internal/daemon/mcp/server.go` - Removed lifecycle dependency
- `internal/daemon/api/deps.go` - Removed HookExecutor
- `internal/daemon/api/server.go` - Added POST /sessions/{id}/approve route
- `internal/daemon/api/sessions.go` - Added Approve handler
- `internal/core/spawn/spawn.go` - Updated to use new template system
- `internal/prompt/prompt.go` - Added new path functions
- `internal/install/install.go` - Added creation of 5 new prompt files
- `internal/cli/sdk/client.go` - Added ApproveSession method
- `internal/cli/tui/kanban/model.go` - Added approve action with 'a' key
- `internal/cli/tui/kanban/keys.go` - Added KeyApprove constant
- `cmd/cortexd/commands/serve.go` - Removed lifecycle executor

### Important Decisions
- Prompts fall back gracefully (e.g., if ticket-system.md missing, uses legacy ticket-agent.md)
- Approve sends prompt via tmux send-keys to agent pane
- ReviewRequest stored in Session.RequestedReviews array (supports multi-repo)
- concludeSession adds ticket_done comment before ending session

### Scope Changes
- None - implemented as specified
