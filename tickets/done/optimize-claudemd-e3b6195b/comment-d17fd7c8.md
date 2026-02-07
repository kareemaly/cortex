---
id: d17fd7c8-e5b8-419d-b94c-88a89c3ab203
author: claude
type: done
created: 2026-02-04T15:23:43.460034Z
---
## Summary

Optimized CLAUDE.md based on best practices by adding 5 new sections while keeping the file concise.

### Changes Made

**New Sections Added:**
1. **Quick Start** (lines 5-11) - Essential 3-command workflow for immediate onboarding
2. **Critical Implementation Notes** (lines 42-47) - Key architectural constraints:
   - HTTP-only communication between clients and daemon
   - Project context headers/env requirements
   - StoreManager as single source of truth
   - Spawn state detection (4 states, 3 modes)
3. **Anti-Patterns** (lines 49-56) - Table of common mistakes:
   - Don't access ticket JSON directly → use SDK client
   - Don't spawn tmux directly → use SpawnSession via API
   - Don't import internal packages in CLI → use HTTP endpoints
4. **Debugging** (lines 58-65) - Troubleshooting reference for common issues

**Updated:**
- **CLI Commands table** - Added 3 missing commands synced from README.md:
  - `cortex daemon status`
  - `cortex upgrade`
  - `cortex eject <path>`

### Files Changed
- `CLAUDE.md` - 36 lines added (110 → 147 lines, within 200-300 target)

### Commit
`13f89ba` - docs: add quick start, implementation notes, anti-patterns, and debugging sections to CLAUDE.md

Pushed to `origin/main`.