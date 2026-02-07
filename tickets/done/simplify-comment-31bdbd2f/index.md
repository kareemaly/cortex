---
id: 31bdbd2f-a2bb-45b5-a031-e01f449d76f1
title: Simplify Comment Model, Remove Reviews, and Restructure Ticket MCP Tools
type: ""
created: 2026-01-28T09:35:30.868394Z
updated: 2026-01-28T10:00:29.3792Z
---
## Summary

Simplify the comment model from 8 types to 4, unify review requests into comments via an `action` field, remove the separate `RequestedReviews` from the session model, remove the `Title` field from comments, and restructure the ticket agent MCP tools to match the new model.

## Current State

**Comment**: `{ID, SessionID, Type (8 types), Title, Content, CreatedAt}`
- Types: scope_change, decision, blocker, progress, question, rejection, general, ticket_done

**ReviewRequest**: Separate struct `{RepoPath, Title, Content, RequestedAt}` stored on `Session.RequestedReviews[]`

**MCP tools** (ticket agent): `readTicket`, `addTicketComment(type, title, content)`, `requestReview(repo_path, title, content)`, `concludeSession(full_report)`

## Target State

### Comment model

```
Comment {
  ID        string
  SessionID string
  Type      CommentType    // review_requested | done | blocker | comment
  Content   string         // markdown; first line should be a plain-text summary
  Action    *CommentAction // optional, only for actionable comments
  CreatedAt time.Time
}

CommentAction {
  Type string  // "git_diff" for now
  Args any     // differs by action type
}
```

- **4 comment types**: `review_requested`, `done`, `blocker`, `comment`
- **Remove** `Title` field — content's first line serves as the summary
- **Add** `Action` field (optional) — structured data for actionable comments
- For `review_requested`: action type is `git_diff`, args contains `repo_path` (string) and optionally `commit` (string)

### Remove from session model

- Remove `RequestedReviews []ReviewRequest` from Session struct
- Remove `ReviewRequest` struct entirely
- Remove `store.AddReviewRequest()` method
- Reviews are now just comments with `type=review_requested`

### MCP tools (ticket agent)

Replace the current 4 tools with:

| Tool | Parameters | Comment Type | Action |
|------|-----------|-------------|--------|
| `addComment(content)` | content: markdown | `comment` | none |
| `addBlocker(content)` | content: markdown summary | `blocker` | none |
| `requestReview(content, repo_path, commit?)` | content: markdown summary, repo_path: string, commit: string (optional) | `review_requested` | `{type: "git_diff", args: {repo_path, commit?}}` |
| `concludeSession(content)` | content: markdown summary | `done` | none |
| `readTicket` | (unchanged) | — | — |

- `requestReview` should still move ticket to review status (existing behavior)
- `concludeSession` should still end the session and move ticket to done (existing behavior)

### API endpoints

- Update `POST /tickets/{id}/comments` to accept new shape (no title, optional action)
- Remove or repurpose `POST /tickets/{id}/reviews` — reviews now go through comments endpoint
- Update `POST /tickets/{id}/conclude` to use new comment shape
- Update all response serialization to reflect new model

## Acceptance Criteria

- [ ] Comment struct has 4 types: review_requested, done, blocker, comment
- [ ] Comment struct has no Title field
- [ ] Comment struct has optional Action field with Type and Args
- [ ] Session struct has no RequestedReviews field
- [ ] ReviewRequest struct is removed
- [ ] MCP tools updated: addComment, addBlocker, requestReview, concludeSession, readTicket
- [ ] requestReview creates a comment with type=review_requested and action={type: git_diff, args: {repo_path, commit?}}
- [ ] requestReview still moves ticket to review status
- [ ] concludeSession creates a comment with type=done
- [ ] API endpoints updated to match new model
- [ ] Existing tests updated, all pass (`make test`)