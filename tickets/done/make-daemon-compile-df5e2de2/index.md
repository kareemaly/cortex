---
id: df5e2de2-99df-4eef-9af8-0930dc1d3edb
title: Make daemon compile with new storage layer
type: work
created: 2026-02-07T08:16:39.865067Z
updated: 2026-02-07T09:18:55.562576Z
---
## Overview

The storage layer packages (`internal/ticket/`, `internal/docs/`, `internal/session/`, `internal/storage/`) have been rewritten to use YAML frontmatter + directory-per-entity. Everything above them (daemon, API, MCP, spawn, TUI, CLI) now fails to compile. This ticket fixes all compilation errors and gets the daemon building again.

## What Changed in Storage Layer

### `internal/ticket/`
- `Ticket` struct now uses `TicketMeta` with YAML tags (no more JSON)
- `Dates` struct removed — replaced with flat `Created`, `Updated`, `Due` fields on Ticket
- `Session` struct removed — moved to `internal/session/`
- `StatusEntry`, `AgentStatus` types removed — moved to `internal/session/`
- `Comment` is now from `internal/storage` (shared type with `Author` instead of `SessionID`)
- Store API changes:
  - `Create()` signature may differ
  - `Get()` now returns `(*Ticket, Status, error)` — loads comments
  - `List()` returns tickets WITHOUT comments (performance)
  - `AddComment()` uses new comment type (author, type, content, action)
  - `SetSession()`, `EndSession()`, `UpdateSessionStatus()` REMOVED
  - `SetDueDate()` / `ClearDueDate()` replaced by `Update()` with due field or dedicated method
- Re-exports shared types as aliases for backward compatibility

### `internal/docs/`
- `Doc` uses `DocMeta` — `Category` field removed from struct (derived from directory path)
- Category is returned by store methods, not embedded in doc
- New `AddComment()` / `ListComments()` capability
- Same directory-per-entity store pattern as tickets

### `internal/session/` (NEW)
- `Session` struct: ID, TicketID, Agent, TmuxWindow, WorktreePath, FeatureBranch, StartedAt, Status, Tool
- `AgentStatus` type: starting, in_progress, idle, waiting_permission, error
- `Store` backed by `.cortex/sessions.json`
- CRUD: `Create`, `Get`, `GetByTicketID`, `UpdateStatus`, `End`, `List`
- Keyed by session ID

### `internal/storage/` (NEW — shared)
- `NotFoundError`, `ValidationError`, `IsNotFound`
- `GenerateSlug(title, fallback)`, `ShortID(id)`, `DirName(title, id)`
- Generic `ParseFrontmatter[T]` / `SerializeFrontmatter[T]`
- `AtomicWriteFile`
- `Comment`, `CommentMeta`, `CommentAction` shared types
- `CreateComment()`, `ListComments()` shared functions

## Scope

### 1. Update `internal/project/config/`
- Add `tickets.path` to config (default: `"tickets"` relative to project root)
- Update `docs.path` default from `.cortex/docs` to `"docs"` relative to project root
- Both support relative (to project root) and absolute paths
- Add `TicketsPath(projectRoot string) string` method mirroring existing `DocsPath()`

### 2. Update `internal/types/` (response types)
- Update `TicketResponse` — remove `Dates` nesting, use flat `Created`/`Updated`/`Due` fields, remove `Session` field
- Update `TicketSummary` — same flat date fields
- Update `CommentResponse` — `Author` instead of `SessionID`, keep action
- Update `DocResponse` — adjust for category-from-path
- Update `DocSummary` — same
- Update conversion functions (`ToTicketResponse`, `ToDocResponse`, etc.)
- Add session response types if needed

### 3. Update `internal/daemon/api/store_manager.go`
- Update `GetStore()` to use new config path (`TicketsPath()` instead of hardcoded `.cortex/tickets`)

### 4. Update `internal/daemon/api/docs_store_manager.go`
- Ensure it uses updated `DocsPath()` with new default

### 5. Create session management in daemon
- Create `SessionManager` (or equivalent) wrapping `session.Store`
- Session store file lives at `{projectPath}/.cortex/sessions.json`
- Wire into API server

### 6. Update `internal/daemon/api/` handlers
- Fix all ticket handlers for new types (no Dates struct, no Session field, flat fields)
- Fix doc handlers for category-from-path
- Fix comment handlers for new Comment type (Author instead of SessionID)
- Add/update session-related endpoints to use session store
- Fix all compilation errors

### 7. Update `internal/daemon/mcp/` — just fix compilation
- Fix type references so MCP tools compile
- Don't worry about perfecting the MCP tool behavior — that's ticket 2c
- Just make it build

### 8. Update `internal/core/spawn/` — just fix compilation
- Fix type references so spawn code compiles
- Don't worry about perfecting session lifecycle — that's ticket 2b
- Just make it build

### 9. Update `internal/cli/` — just fix compilation
- Fix TUI and CLI code that references old types
- Don't worry about perfecting TUI rendering — that may be a later ticket
- Just make it build

## Goals
- `make build` passes
- `make lint` passes
- `make test` passes (at minimum the storage package tests, ideally all unit tests)
- No old type references remaining (`Dates`, `ticket.Session`, `ticket.AgentStatus`, `ticket.StatusEntry`)

## Key Constraints
- **Breaking changes fine** — single user
- **No tech debt** — clean updates, not hacks to make it compile
- **Width over depth** — touch everything, fix types, but don't rewrite business logic for spawn/MCP (those are follow-up tickets)

## Branch
Working on `feat/frontmatter-storage` branch.