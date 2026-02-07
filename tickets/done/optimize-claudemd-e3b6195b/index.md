---
id: e3b6195b-7662-46c2-a4c9-09cc2756abeb
title: Optimize CLAUDE.md based on best practices analysis
type: work
created: 2026-02-04T14:20:19.326426Z
updated: 2026-02-04T15:23:43.461072Z
---
## Context

Analysis of CLAUDE.md files across multiple projects revealed patterns for effective AI agent guidance. Current Cortex CLAUDE.md is solid but has gaps.

## Current Strengths
- Key Paths table ✓
- MCP Tools section ✓
- Agent Workflow section ✓
- Architecture diagram ✓

## Changes Required

### 1. Add "Critical Implementation Notes" section
Non-obvious pitfalls that cause rework:
- HTTP-only communication (no direct filesystem access from clients)
- Project context via `X-Cortex-Project` header
- Spawn orchestration state detection (normal/active/orphaned/ended)
- StoreManager is single source of truth

### 2. Add "Don't Use X, Use Y" section
Anti-patterns specific to daemon architecture:
- Don't access ticket store directly → Use SDK client
- Don't spawn tmux directly → Use spawn orchestrator
- Don't read project config from clients → Use API endpoints

### 3. Add "Debugging" section by symptom
Common issues:
- "Daemon not responding" → Check port 4200, `cortex daemon status`
- "Ticket not found" → Verify X-Cortex-Project header
- "Session won't spawn" → Check tmux state, orphaned sessions

### 4. Sync CLI commands with README.md
Ensure both files cover the same command set (currently minor gaps).

### 5. Add Quick Start commands (5 essential)
Group by activity: build, test, lint, run daemon, run CLI.

## Guidance
- Keep high-level: architecture, paths, patterns
- Detailed implementation docs → `docs/` or code comments
- Target 200-300 lines