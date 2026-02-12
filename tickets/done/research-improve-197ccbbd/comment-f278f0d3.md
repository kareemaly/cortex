---
id: f278f0d3-8a90-4140-9a5b-e0e34a90abe2
author: claude
type: done
created: 2026-02-09T16:36:52.180922Z
---
## Summary

Researched and applied improvements to the architect SYSTEM.md prompt based on Claude prompting best practices, Claude Code system prompt patterns, and documented behavioral issues.

### Changes Made

**File modified:** `internal/install/defaults/claude-code/prompts/architect/SYSTEM.md` (commit 70767ff, merged to main and pushed)

**Key additions (~73 lines added):**

1. **`<ticket_quality>` section** — Core "WHAT not HOW" principle with include/exclude lists. Explains why assumptions are harmful ("you have not read the code") and provides an escape hatch (explore agent for verified facts).
2. **Good/bad ticket examples** — Paired `<example_bad>` and `<example_good>` showing the contrast between assumption-laden and requirements-focused tickets.
3. **Ticket type guidance** — work, debug, research, chore with descriptions.
4. **Scoping heuristic** — "Each ticket completable by one agent in one session."
5. **Explore-first workflow** — Spawn explore agent before writing tickets that need technical accuracy.
6. **Docs workflow** — Mention of `createDoc` for persisting decisions.
7. **Extended "no time estimates"** — Now explicitly covers tickets too.
8. **Minor wording fix** — `<do_not_act_before_instructions>` now says "discuss scope and requirements first."

### Decisions Made
- Examples kept inline (SYSTEM.md must be self-contained)
- Strict "never include file paths" default with explore-agent escape hatch
- Cross-project guidance omitted (meta agent's job)
- No rigid ticket template (examples > templates)