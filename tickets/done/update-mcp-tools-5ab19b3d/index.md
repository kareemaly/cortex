---
id: 5ab19b3d-e3d2-4a96-8776-ef1b6e978331
title: Update MCP tools for new storage layer
type: work
created: 2026-02-07T09:32:42.446385Z
updated: 2026-02-07T09:59:25.817202Z
---
## Overview

Update all MCP tools (`internal/daemon/mcp/`) to properly work with the new frontmatter + directory-per-entity storage layer. Ticket 2a made them compile, but the tools need to be updated to expose new fields, add new tools, and ensure correct behavior.

## Architect Tools — Changes

### Enriched existing tools

**`listTickets`**
- Add `tag` filter parameter (case-insensitive, like docs)
- Enrich summary output to include: `type`, `tags`, `due`, `created`, `updated` (currently only `id` + `title`)

**`readTicket`**
- Output: flat dates (`created`, `updated`, `due`), no session field
- Include `tags` in output
- Include comments (with `author` field, not `session_id`)

**`createTicket`**
- Add `tags` parameter (optional `[]string`)

**`updateTicket`**
- Add `tags` parameter (optional `*[]string`, full replacement)

**`readDoc`**
- Include comments in output (docs now support comments)

**`addTicketComment`**
- Uses `author` field — default to `"architect"` for architect session

### New tools

**`addDocComment`**
- Same pattern as `addTicketComment` but for docs
- Input: `id` (required), `type` (required), `content` (required), `project_path` (optional)
- Author defaults to `"architect"`
- Cross-project support

**`listSessions`**
- List all active sessions
- Input: `project_path` (optional)
- Output: array of session objects, each with:
  - `session_id`
  - `ticket_id`
  - `ticket_title` (resolve from ticket store)
  - `agent`
  - `tmux_window`
  - `started_at`
  - `status`
  - `tool`

## Ticket Agent Tools — Changes

All 5 tools stay, updated types:

**`readTicket`** — same output changes as architect (flat dates, tags, comments with author)

**`addComment`** — author auto-set to agent name from session

**`addBlocker`** — author auto-set to agent name from session

**`requestReview`** — author auto-set, action preserved

**`concludeSession`** — uses session store to end session

## MCP Types (`types.go`)

### Update `TicketOutput`
```go
type TicketOutput struct {
    ID         string          `json:"id"`
    Type       string          `json:"type"`
    Title      string          `json:"title"`
    Body       string          `json:"body"`
    Tags       []string        `json:"tags,omitempty"`
    References []string        `json:"references,omitempty"`
    Status     string          `json:"status"`
    Created    time.Time       `json:"created"`
    Updated    time.Time       `json:"updated"`
    Due        *time.Time      `json:"due,omitempty"`
    Comments   []CommentOutput `json:"comments"`
}
```

### Update `TicketSummary`
```go
type TicketSummary struct {
    ID      string     `json:"id"`
    Title   string     `json:"title"`
    Type    string     `json:"type"`
    Tags    []string   `json:"tags,omitempty"`
    Due     *time.Time `json:"due,omitempty"`
    Created string     `json:"created"`
    Updated string     `json:"updated"`
}
```

### Update `CommentOutput`
```go
type CommentOutput struct {
    ID      string          `json:"id"`
    Author  string          `json:"author"`
    Type    string          `json:"type"`
    Content string          `json:"content"`
    Action  *CommentAction  `json:"action,omitempty"`
    Created time.Time       `json:"created"`
}
```

### New `SessionOutput` (for listSessions)
```go
type SessionListItem struct {
    SessionID   string    `json:"session_id"`
    TicketID    string    `json:"ticket_id"`
    TicketTitle string    `json:"ticket_title"`
    Agent       string    `json:"agent"`
    TmuxWindow  string    `json:"tmux_window"`
    StartedAt   time.Time `json:"started_at"`
    Status      string    `json:"status"`
    Tool        *string   `json:"tool,omitempty"`
}
```

### Update `DocOutput`
Add comments field:
```go
type DocOutput struct {
    ID         string          `json:"id"`
    Title      string          `json:"title"`
    Category   string          `json:"category"`
    Tags       []string        `json:"tags,omitempty"`
    References []string        `json:"references,omitempty"`
    Body       string          `json:"body"`
    Created    string          `json:"created"`
    Updated    string          `json:"updated"`
    Comments   []CommentOutput `json:"comments,omitempty"`
}
```

## Input Types

### New/Updated inputs

**`ListTicketsInput`** — add `tag` field:
```go
Tag string `json:"tag,omitempty"`
```

**`CreateTicketInput`** — add `tags` field:
```go
Tags []string `json:"tags,omitempty"`
```

**`UpdateTicketInput`** — add `tags` field:
```go
Tags *[]string `json:"tags,omitempty"`
```

**`AddDocCommentInput`** (new):
```go
type AddDocCommentInput struct {
    ID          string `json:"id"`
    Type        string `json:"type"`
    Content     string `json:"content"`
    ProjectPath string `json:"project_path,omitempty"`
}
```

**`ListSessionsInput`** (new):
```go
type ListSessionsInput struct {
    ProjectPath string `json:"project_path,omitempty"`
}
```

## Handler Implementation Notes

- Architect comment tools: set `author` to `"architect"`
- Ticket agent comment tools: look up agent name from session store using the ticket ID, use that as `author`
- `listSessions`: call `sessionStore.List()`, then for each session resolve ticket title from ticket store
- `listTickets` with tag filter: pass through to SDK/API (the ticket store `List` may need a tag filter added, or filter in the handler)
- Conversion functions: update all `ticketToOutput`, `docToOutput` etc. to map new fields

## Tests

- Update existing MCP tests for new types
- Add tests for `addDocComment`
- Add tests for `listSessions`
- Add tests for `listTickets` with tag filter
- Verify all tools return correct shapes

## Goals

- All MCP tools work correctly with new storage
- New tools (`addDocComment`, `listSessions`) functional
- `listTickets` returns enriched summaries with type, tags, due
- `make build && make lint && make test` pass

## Branch

Working on `feat/frontmatter-storage` branch.