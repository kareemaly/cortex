---
id: 388d881d-14c1-4a6a-a399-3bfaa89bd8d2
title: Migration script for ~/kesc to new storage format
type: work
created: 2026-02-07T10:44:55.932619Z
updated: 2026-02-07T10:55:33.388124Z
---
## Overview

Write a one-time migration script that converts the existing project at `~/kesc` from the old JSON ticket storage (inside `.cortex/tickets/`) to the new YAML frontmatter + directory-per-entity format (at project root `tickets/` and `docs/`).

## What to migrate

### Tickets (`.cortex/tickets/{status}/*.json` → `tickets/{status}/{slug}-{shortid}/index.md`)

For each JSON ticket file:
1. Parse the JSON
2. Create entity directory: `tickets/{status}/{slug}-{shortid}/`
3. Write `index.md` with YAML frontmatter:
   ```yaml
   ---
   id: <full UUID>
   title: <title>
   type: <type>
   tags: []
   references: <references if any>
   due: <due_date if any>
   created: <dates.created>
   updated: <dates.updated>
   ---
   <body>
   ```
4. For each comment in the JSON `comments` array, write `comment-{shortid}.md`:
   ```yaml
   ---
   id: <comment UUID>
   author: <map session_id to agent name, or "unknown">
   type: <comment type>
   created: <created_at>
   action:
     type: <action.type>
     args: <action.args>
   ---
   <content>
   ```
5. Drop session data (ephemeral, not migrated)
6. Drop lifecycle dates (progress, reviewed, done) — derived from comments now

### Docs (`.cortex/docs/{category}/*.md` → `docs/{category}/{slug}-{shortid}/index.md`)

For each existing doc file:
1. Parse the frontmatter (already YAML)
2. Create entity directory: `docs/{category}/{slug}-{shortid}/`
3. Write `index.md` — same frontmatter but remove `category` field (now derived from path)
4. No comments to migrate (docs didn't have comments before)

## Implementation

Write this as a standalone Go script in `cmd/migrate/main.go` (or a shell script — whatever is quickest and most reliable). It should:

1. Take the project path as an argument (default `~/kesc`)
2. Read all tickets from `.cortex/tickets/{status}/`
3. Read all docs from `.cortex/docs/{category}/`
4. Create new directory structure at project root
5. Write all migrated files
6. Print summary (X tickets migrated, Y docs migrated, Z comments migrated)
7. Do NOT delete old files — leave them for manual cleanup after verification

## Mapping notes

- `session_id` on old comments → map to `"claude"` as author (all sessions were claude agents)
- `comment.type` values: the old `"ticket_done"` type should map to `"done"`, `"ticket_comment"` to `"comment"`. Check actual values in the JSON files.
- Old `dates.due_date` → new `due`
- Tags: old tickets don't have tags, so use empty `tags: []`
- References: preserve as-is

## Verification

After running:
- New `tickets/` directory should have correct structure
- New `docs/` directory should have correct structure
- `cortexd` should be able to read all migrated tickets and docs
- Comment count should match

## Branch

Working on `main` branch.