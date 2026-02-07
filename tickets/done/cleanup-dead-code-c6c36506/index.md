---
id: c6c36506-9b16-4913-90cd-67acaa154d57
title: Cleanup dead code and update documentation
type: chore
created: 2026-02-07T10:14:30.539438Z
updated: 2026-02-07T10:22:39.812491Z
---
## Overview

Final cleanup after the frontmatter + directory-per-entity storage migration. Remove all dead code, unused types, stale compatibility layers, and update documentation to reflect the new architecture.

## Scope

### 1. Dead code removal

Search for and remove:
- Any remaining references to old types that are no longer used: `DatesResponse`, `StatusEntryResponse`, old `SessionResponse` shapes
- Unused imports across all files
- Commented-out code from the migration
- Any backward-compatibility aliases in `internal/ticket/` or `internal/docs/` that are no longer referenced
- Old frontmatter/slug/error files that were moved to `internal/storage/` but may still have stubs
- Any unused helper functions or methods that existed for the old storage model

### 2. Verify no stale patterns

Run these checks and fix any hits:
- `\.Dates\.` — old nested dates access
- `ticket\.Session` — old embedded session
- `ticket\.AgentStatus` — moved to session package
- `ticket\.StatusEntry` — moved to session package
- `SessionID` on comments — replaced by `Author`
- `\.cortex/tickets` or `.cortex/docs` hardcoded paths (should be configurable now)
- `StateEnded` — removed state

### 3. Update CLAUDE.md

Update the project documentation to reflect:
- New storage format (YAML frontmatter + directory-per-entity)
- New directory layout (tickets/ and docs/ at project root by default)
- New packages: `internal/storage/` (shared), `internal/session/` (ephemeral sessions)
- Updated key paths table
- Session model: independent, ephemeral, `.cortex/sessions.json`
- Removed concepts: `Dates` struct, embedded sessions on tickets, `StateEnded`
- New MCP tools: `addDocComment`, `listSessions`
- Updated MCP tool descriptions where fields changed
- Config: `tickets.path` and `docs.path` in cortex.yaml

### 4. Check test coverage

- Verify no tests reference old types or patterns
- Remove any skipped/commented tests from migration
- Ensure all test files compile and pass

## Goals

- Zero dead code from the migration
- CLAUDE.md accurately describes the current architecture
- `make build && make lint && make test` pass
- Clean `git diff` — no leftover debug prints, TODOs, or migration artifacts

## Branch

Working on `feat/frontmatter-storage` branch.