---
id: ff52153e-3a87-4702-ad8e-a17e80cd05e5
title: Persist architect concludeSession summary as a doc
type: work
tags:
    - architect
    - docs
    - conclude
created: 2026-02-07T12:36:12.409317Z
updated: 2026-02-07T12:40:36.180309Z
---
## Problem

When an architect calls `concludeSession`, the `content` parameter (summary of work done, decisions made, files changed) is discarded. The ticket agent's conclude persists its content as a "done" comment, but the architect path throws it away.

## Solution

In the architect conclude handler (`internal/daemon/api/architect.go`, `Conclude` method), after ending the session, create a doc with:

- **Category:** `sessions`
- **Title:** `"Architect Session — {timestamp}"` (e.g., `"Architect Session — 2026-02-07T12:21Z"`)
- **Body:** The `content` string from the conclude request
- **Tags:** `["architect", "session-summary"]`

This gives a persistent audit trail of architect session summaries, queryable via `listDocs(category: "sessions")`.

## Scope

- Only the architect conclude path — ticket agent already persists via done comment
- The doc creation should be best-effort (log warning on failure, don't fail the conclude)
- Use the existing docs store manager to create the doc

## Key Files

- `internal/daemon/api/architect.go` — Conclude handler (lines ~253-302)
- `internal/daemon/api/docs_store_manager.go` — DocsStoreManager for doc creation